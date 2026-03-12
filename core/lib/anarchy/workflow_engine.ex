defmodule Anarchy.WorkflowEngine do
  @moduledoc """
  CE (Compound Engineering) loop state machine.

  Manages the lifecycle of a single task through the CE workflow:
    :idle → :planning → :plan_reviewing → :working → :ce_reviewing
      → :code_reviewing → :compounding → :completed

  Critical findings during review stages roll back to :working.
  Plan review revisions roll back to :planning.
  """

  use GenStateMachine, callback_mode: [:handle_event_function, :state_enter]

  require Logger

  alias Anarchy.{AgentMail, Notifications, Projects, RoleLoader, SessionManager}
  alias Anarchy.Schemas.Task, as: TaskSchema

  @type ce_state ::
          :idle
          | :planning
          | :plan_reviewing
          | :working
          | :ce_reviewing
          | :code_reviewing
          | :compounding
          | :completed
          | :failed

  defmodule Data do
    @moduledoc false
    defstruct [
      :task,
      :project_id,
      :workspace_path,
      :current_worker_pid,
      :current_worker_ref,
      :session_id,
      :learnings,
      :feedback,
      :plan_output,
      attempt: 0,
      max_attempts: 3,
      ce_review_results: [],
      started_at: nil
    ]
  end

  # --- Public API ---

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    task = Keyword.fetch!(opts, :task)
    name = Keyword.get(opts, :name, via_name(task.id))
    GenStateMachine.start_link(__MODULE__, opts, name: name)
  end

  @spec current_state(GenServer.server()) :: {ce_state(), Data.t()}
  def current_state(server) do
    GenStateMachine.call(server, :current_state)
  end

  @spec trigger(GenServer.server(), atom()) :: :ok
  def trigger(server, event) do
    GenStateMachine.cast(server, event)
  end

  # --- Callbacks ---

  @impl true
  def init(opts) do
    task = Keyword.fetch!(opts, :task)
    workspace_path = Keyword.get(opts, :workspace_path)

    data = %Data{
      task: task,
      project_id: task.project_id,
      workspace_path: workspace_path,
      started_at: DateTime.utc_now()
    }

    actions = [{:next_event, :internal, :start}]
    {:ok, :idle, data, actions}
  end

  @impl true
  def terminate(_reason, _state, data) do
    # Kill worker process to prevent orphans when WorkflowEngine crashes
    if data.current_worker_pid && Process.alive?(data.current_worker_pid) do
      Process.exit(data.current_worker_pid, :shutdown)
    end

    :ok
  end

  # --- State: :idle ---

  @impl true
  def handle_event(:enter, _old_state, :idle, _data) do
    :keep_state_and_data
  end

  def handle_event(:internal, :start, :idle, data) do
    Logger.info("CE loop starting for task_id=#{data.task.id} title=#{data.task.title}")
    {:next_state, :planning, data}
  end

  # --- State: :planning ---

  def handle_event(:enter, _old_state, :planning, data) do
    Logger.info("CE loop entering :planning for task_id=#{data.task.id}")
    {:keep_state_and_data, [{:state_timeout, 0, :run_plan}]}
  end

  def handle_event(:state_timeout, :run_plan, :planning, data) do
    case spawn_role_worker(:developer, data, &plan_prompt/1) do
      {:ok, pid, ref, session_id} ->
        {:keep_state, %{data | current_worker_pid: pid, current_worker_ref: ref, session_id: session_id}}

      {:error, reason} ->
        Logger.error("Failed to start planning worker: #{inspect(reason)}")
        {:next_state, :failed, %{data | feedback: reason}}
    end
  end

  def handle_event(:info, {:worker_complete, :normal, output}, :planning, data) do
    demonitor_worker(data)
    {:next_state, :plan_reviewing, %{data | plan_output: output, current_worker_pid: nil, current_worker_ref: nil}}
  end

  def handle_event(:info, {:worker_complete, reason, _output}, :planning, data) do
    demonitor_worker(data)
    handle_worker_failure(data, :planning, reason)
  end

  # --- State: :plan_reviewing ---

  def handle_event(:enter, _old_state, :plan_reviewing, data) do
    Logger.info("CE loop entering :plan_reviewing for task_id=#{data.task.id}")
    {:keep_state_and_data, [{:state_timeout, 0, :run_plan_review}]}
  end

  def handle_event(:state_timeout, :run_plan_review, :plan_reviewing, data) do
    case spawn_role_worker(:plan_reviewer, data, &plan_review_prompt/1) do
      {:ok, pid, ref, session_id} ->
        {:keep_state, %{data | current_worker_pid: pid, current_worker_ref: ref, session_id: session_id}}

      {:error, reason} ->
        Logger.error("Failed to start plan review worker: #{inspect(reason)}")
        {:next_state, :failed, %{data | feedback: reason}}
    end
  end

  def handle_event(:info, {:worker_complete, :normal, output}, :plan_reviewing, data) do
    demonitor_worker(data)

    case classify_review_result(output) do
      :approved ->
        {:next_state, :working, %{data | current_worker_pid: nil, current_worker_ref: nil}}

      :revision_needed ->
        Logger.info("Plan review requested revision for task_id=#{data.task.id}")
        {:next_state, :planning, %{data | feedback: output, current_worker_pid: nil, current_worker_ref: nil}}
    end
  end

  def handle_event(:info, {:worker_complete, reason, _output}, :plan_reviewing, data) do
    demonitor_worker(data)
    handle_worker_failure(data, :plan_reviewing, reason)
  end

  # --- State: :working ---

  def handle_event(:enter, _old_state, :working, data) do
    Logger.info("CE loop entering :working for task_id=#{data.task.id}")
    {:keep_state_and_data, [{:state_timeout, 0, :run_work}]}
  end

  def handle_event(:state_timeout, :run_work, :working, data) do
    case spawn_role_worker(:developer, data, &work_prompt/1) do
      {:ok, pid, ref, session_id} ->
        {:keep_state, %{data | current_worker_pid: pid, current_worker_ref: ref, session_id: session_id}}

      {:error, reason} ->
        Logger.error("Failed to start work worker: #{inspect(reason)}")
        {:next_state, :failed, %{data | feedback: reason}}
    end
  end

  def handle_event(:info, {:worker_complete, :normal, _output}, :working, data) do
    demonitor_worker(data)
    {:next_state, :ce_reviewing, %{data | current_worker_pid: nil, current_worker_ref: nil}}
  end

  def handle_event(:info, {:worker_complete, reason, _output}, :working, data) do
    demonitor_worker(data)
    handle_worker_failure(data, :working, reason)
  end

  # --- State: :ce_reviewing ---

  def handle_event(:enter, _old_state, :ce_reviewing, data) do
    Logger.info("CE loop entering :ce_reviewing for task_id=#{data.task.id}")
    {:keep_state_and_data, [{:state_timeout, 0, :run_ce_review}]}
  end

  def handle_event(:state_timeout, :run_ce_review, :ce_reviewing, data) do
    case spawn_role_worker(:ce_reviewer, data, &ce_review_prompt/1) do
      {:ok, pid, ref, session_id} ->
        {:keep_state, %{data | current_worker_pid: pid, current_worker_ref: ref, session_id: session_id}}

      {:error, reason} ->
        Logger.error("Failed to start CE review worker: #{inspect(reason)}")
        {:next_state, :failed, %{data | feedback: reason}}
    end
  end

  def handle_event(:info, {:worker_complete, :normal, output}, :ce_reviewing, data) do
    demonitor_worker(data)

    case classify_review_result(output) do
      :approved ->
        {:next_state, :code_reviewing, %{data | ce_review_results: [output | data.ce_review_results], current_worker_pid: nil, current_worker_ref: nil}}

      :revision_needed ->
        Logger.info("CE review found critical issues for task_id=#{data.task.id}; rolling back to :working")
        Notifications.notify(:critical_found, %{task: data.task, count: 1})
        data = %{data | feedback: output, ce_review_results: [output | data.ce_review_results], current_worker_pid: nil, current_worker_ref: nil}
        {:next_state, :working, data}
    end
  end

  def handle_event(:info, {:worker_complete, reason, _output}, :ce_reviewing, data) do
    demonitor_worker(data)
    handle_worker_failure(data, :ce_reviewing, reason)
  end

  # --- State: :code_reviewing ---

  def handle_event(:enter, _old_state, :code_reviewing, data) do
    Logger.info("CE loop entering :code_reviewing for task_id=#{data.task.id}")
    {:keep_state_and_data, [{:state_timeout, 0, :run_code_review}]}
  end

  def handle_event(:state_timeout, :run_code_review, :code_reviewing, data) do
    case spawn_role_worker(:code_reviewer, data, &code_review_prompt/1) do
      {:ok, pid, ref, session_id} ->
        {:keep_state, %{data | current_worker_pid: pid, current_worker_ref: ref, session_id: session_id}}

      {:error, reason} ->
        Logger.error("Failed to start code review worker: #{inspect(reason)}")
        {:next_state, :failed, %{data | feedback: reason}}
    end
  end

  def handle_event(:info, {:worker_complete, :normal, output}, :code_reviewing, data) do
    demonitor_worker(data)

    case classify_review_result(output) do
      :approved ->
        {:next_state, :compounding, %{data | current_worker_pid: nil, current_worker_ref: nil}}

      :revision_needed ->
        Logger.info("Code review found critical issues for task_id=#{data.task.id}; rolling back to :working")
        {:next_state, :working, %{data | feedback: output, current_worker_pid: nil, current_worker_ref: nil}}
    end
  end

  def handle_event(:info, {:worker_complete, reason, _output}, :code_reviewing, data) do
    demonitor_worker(data)
    handle_worker_failure(data, :code_reviewing, reason)
  end

  # --- State: :compounding ---

  def handle_event(:enter, _old_state, :compounding, data) do
    Logger.info("CE loop entering :compounding for task_id=#{data.task.id}")
    {:keep_state_and_data, [{:state_timeout, 0, :run_compound}]}
  end

  def handle_event(:state_timeout, :run_compound, :compounding, data) do
    case spawn_role_worker(:developer, data, &compound_prompt/1) do
      {:ok, pid, ref, session_id} ->
        {:keep_state, %{data | current_worker_pid: pid, current_worker_ref: ref, session_id: session_id}}

      {:error, reason} ->
        Logger.error("Failed to start compound worker: #{inspect(reason)}")
        {:next_state, :completed, %{data | feedback: reason}}
    end
  end

  def handle_event(:info, {:worker_complete, :normal, output}, :compounding, data) do
    demonitor_worker(data)
    Logger.info("CE loop completed for task_id=#{data.task.id}")
    learnings_text = extract_text(output)
    persist_learnings(data, learnings_text)
    {:next_state, :completed, %{data | learnings: learnings_text, current_worker_pid: nil, current_worker_ref: nil}}
  end

  def handle_event(:info, {:worker_complete, reason, _output}, :compounding, data) do
    demonitor_worker(data)
    Logger.warning("Compound step failed for task_id=#{data.task.id}: #{inspect(reason)}; marking completed anyway")
    {:next_state, :completed, %{data | current_worker_pid: nil, current_worker_ref: nil}}
  end

  # --- State: :completed ---

  def handle_event(:enter, _old_state, :completed, data) do
    Logger.info("CE loop completed for task_id=#{data.task.id}")
    update_task_status(data.task, :completed)
    Notifications.notify(:task_completed, %{task: data.task})
    finalize_session(data.session_id, :completed)
    :keep_state_and_data
  end

  # --- State: :failed ---

  def handle_event(:enter, _old_state, :failed, data) do
    Logger.error("CE loop failed for task_id=#{data.task.id}: #{inspect(data.feedback)}")
    update_task_status(data.task, :failed)
    Notifications.notify(:agent_failed, %{task: data.task, reason: data.feedback})
    finalize_session(data.session_id, :failed)
    :keep_state_and_data
  end

  # --- Common handlers ---

  def handle_event({:call, from}, :current_state, state, data) do
    {:keep_state_and_data, [{:reply, from, {state, data}}]}
  end

  def handle_event(:cast, :cancel, _state, data) do
    if data.current_worker_pid && Process.alive?(data.current_worker_pid) do
      Process.exit(data.current_worker_pid, :shutdown)
    end

    {:next_state, :failed, %{data | feedback: :cancelled}}
  end

  def handle_event(:info, {:DOWN, ref, :process, _pid, reason}, _state, %{current_worker_ref: ref} = data) do
    output = nil
    send(self(), {:worker_complete, reason, output})
    {:keep_state, data}
  end

  def handle_event(:info, {:DOWN, _ref, :process, _pid, _reason}, _state, _data) do
    :keep_state_and_data
  end

  # Catch-all for unexpected events
  def handle_event(event_type, event_content, state, _data) do
    Logger.debug("WorkflowEngine ignoring #{inspect(event_type)} #{inspect(event_content)} in state #{state}")
    :keep_state_and_data
  end

  # --- Private helpers ---

  defp spawn_role_worker(role, data, prompt_fn) do
    # Complete previous session before starting new one
    if data.session_id, do: finalize_session(data.session_id, :completed)

    caller = self()
    session_id = "ce-#{data.task.id}-#{role}-#{System.unique_integer([:positive])}"

    {pid, ref} =
      spawn_monitor(fn ->
        try do
          prompt = prompt_fn.(data)
          result = RoleLoader.execute_role(role, data.task, data.workspace_path, prompt)
          send(caller, {:worker_complete, :normal, result})
        catch
          kind, reason ->
            send(caller, {:worker_complete, :error, {kind, reason, __STACKTRACE__}})
        end
      end)

    SessionManager.create_session(%{
      task_id: data.task.id,
      project_id: data.project_id,
      agent_type: RoleLoader.runtime_name(role),
      session_id: session_id,
      role_prompt_path: RoleLoader.role_path(role),
      workspace_path: data.workspace_path,
      status: "active"
    })

    {:ok, pid, ref, session_id}
  rescue
    error -> {:error, error}
  end

  defp demonitor_worker(%{current_worker_ref: ref}) when is_reference(ref) do
    Process.demonitor(ref, [:flush])
  end

  defp demonitor_worker(_data), do: :ok

  defp handle_worker_failure(data, state, reason) do
    data = %{data | attempt: data.attempt + 1, current_worker_pid: nil, current_worker_ref: nil}

    if data.attempt >= data.max_attempts do
      Logger.error("CE loop max attempts reached for task_id=#{data.task.id} in state #{state}")
      {:next_state, :failed, %{data | feedback: reason}}
    else
      Logger.warning("CE loop worker failed for task_id=#{data.task.id} in #{state}: #{inspect(reason)}; retrying (attempt #{data.attempt})")
      {:repeat_state, data}
    end
  end

  # Handle {:ok, text} tuples from run_once/1
  defp classify_review_result({:ok, text}) when is_binary(text) and text != "" do
    classify_review_result(text)
  end

  defp classify_review_result({:error, _reason}), do: :revision_needed

  defp classify_review_result(output) when is_binary(output) and output != "" do
    lower = String.downcase(output)

    cond do
      String.contains?(lower, "critical") -> :revision_needed
      String.contains?(lower, "revision") -> :revision_needed
      String.contains?(lower, "reject") -> :revision_needed
      String.contains?(lower, "approved") -> :approved
      String.contains?(lower, "lgtm") -> :approved
      true -> :revision_needed
    end
  end

  # :ok is NOT a valid review payload — fail-closed
  defp classify_review_result(:ok), do: :revision_needed
  defp classify_review_result(nil), do: :revision_needed
  defp classify_review_result(_output), do: :revision_needed

  defp update_task_status(%TaskSchema{id: task_id}, status) when is_atom(status) do
    Anarchy.Tracker.update_task_state(task_id, Atom.to_string(status))
  rescue
    error ->
      Logger.error("Failed to update task status for #{task_id}: #{inspect(error)}")
      :ok
  end

  defp update_task_status(_task, _status), do: :ok

  defp finalize_session(nil, _status), do: :ok

  defp finalize_session(session_id, :failed) do
    SessionManager.fail_session(session_id)
  rescue
    error ->
      Logger.error("Failed to finalize session #{session_id} as failed: #{inspect(error)}")
      :ok
  end

  defp finalize_session(session_id, _status) do
    SessionManager.complete_session(session_id)
  rescue
    error ->
      Logger.error("Failed to finalize session #{session_id}: #{inspect(error)}")
      :ok
  end

  defp persist_learnings(data, learnings_text) when is_binary(learnings_text) and learnings_text != "" do
    # Save to task record in DB
    case Projects.get_task(data.task.id) do
      nil -> :ok
      task -> Projects.update_task(task, %{learnings: learnings_text})
    end

    # Write to docs/solutions/ in workspace if available
    if data.workspace_path do
      solutions_dir = Path.join(data.workspace_path, "docs/solutions")
      File.mkdir_p(solutions_dir)
      # Sanitize task.id to prevent path traversal
      safe_id = data.task.id |> to_string() |> String.replace(~r/[^a-zA-Z0-9_\-]/, "_")
      filename = "#{safe_id}-learnings.md"
      path = Path.join(solutions_dir, filename)

      content = """
      # Learnings: #{data.task.title}

      Task ID: #{data.task.id}
      Date: #{DateTime.utc_now() |> DateTime.to_iso8601()}

      #{learnings_text}
      """

      File.write(path, content)
    end

    :ok
  rescue
    error ->
      Logger.error("Failed to persist learnings for task #{data.task.id}: #{inspect(error)}")
      :ok
  end

  defp persist_learnings(_data, _learnings), do: :ok

  defp extract_text({:ok, text}) when is_binary(text), do: text
  defp extract_text({:error, _reason}), do: ""
  defp extract_text(output) when is_binary(output), do: output
  defp extract_text(%{"output" => text}) when is_binary(text), do: text
  defp extract_text(%{output: text}) when is_binary(text), do: text
  defp extract_text(:ok), do: ""
  defp extract_text(other), do: inspect(other)

  defp inject_mail_context(prompt, agent_role, project_id) do
    unread = AgentMail.inbox(agent_role, unread_only: true, project_id: project_id)

    case unread do
      [] ->
        prompt

      messages ->
        mail_text =
          Enum.map_join(messages, "\n", fn m ->
            "[#{m.from_agent}] #{m.subject}: #{m.body}"
          end)

        prompt <> "\n\n--- Unread Messages ---\n" <> mail_text
    end
  rescue
    _ -> prompt
  end

  defp via_name(task_id), do: {:global, {__MODULE__, task_id}}

  # --- Prompt builders ---

  defp plan_prompt(data) do
    feedback_section =
      if data.feedback do
        "\n\nPrevious feedback to address:\n#{inspect(data.feedback)}"
      else
        ""
      end

    learnings_section =
      if data.learnings do
        "\n\nLearnings from previous tasks:\n#{data.learnings}"
      else
        ""
      end

    base = """
    Create an implementation plan for the following task.

    Task: #{data.task.title}
    Description: #{data.task.description || "No description provided."}
    #{feedback_section}#{learnings_section}

    Output a structured plan with:
    1. Step-by-step implementation approach
    2. Files to create or modify
    3. Key design decisions
    4. Test strategy
    """

    inject_mail_context(base, "developer", data.project_id)
  end

  defp plan_review_prompt(data) do
    base = """
    Review this implementation plan for structural soundness and direction.
    This is a lightweight review — do NOT review code.

    Task: #{data.task.title}
    Plan: #{inspect(data.plan_output)}

    Check:
    - Is the approach reasonable?
    - Are there missing steps?
    - Are there architectural concerns?

    Respond with APPROVED if acceptable, or describe revisions needed.
    """

    inject_mail_context(base, "plan_reviewer", data.project_id)
  end

  defp work_prompt(data) do
    feedback_section =
      if data.feedback do
        "\n\nReview feedback to address:\n#{inspect(data.feedback)}"
      else
        ""
      end

    base = """
    Implement the following task according to the plan.

    Task: #{data.task.title}
    Description: #{data.task.description || "No description provided."}
    Plan: #{inspect(data.plan_output)}
    #{feedback_section}
    """

    inject_mail_context(base, "developer", data.project_id)
  end

  defp ce_review_prompt(data) do
    base = """
    Perform a Compound Engineering review of the recent changes.

    Task: #{data.task.title}

    Review for:
    1. Security vulnerabilities (auth, input validation, injection)
    2. Performance issues (N+1 queries, unnecessary allocations, O(n²))
    3. Architecture compliance (design principles, separation of concerns)

    If any CRITICAL issues found, respond with "CRITICAL:" followed by the issues.
    If only minor issues, respond with "APPROVED" and list suggestions.
    """

    inject_mail_context(base, "ce_reviewer", data.project_id)
  end

  defp code_review_prompt(data) do
    base = """
    Final code review for the following task.

    Task: #{data.task.title}

    Review for:
    1. Code correctness
    2. Test coverage
    3. Documentation
    4. Style consistency

    If any CRITICAL issues found, respond with "CRITICAL:" followed by the issues.
    If acceptable, respond with "APPROVED".
    """

    inject_mail_context(base, "code_reviewer", data.project_id)
  end

  defp compound_prompt(data) do
    base = """
    Document what was learned from implementing this task.

    Task: #{data.task.title}
    CE Review Results: #{inspect(data.ce_review_results)}

    Write a concise learning document covering:
    1. Key decisions made and why
    2. Problems encountered and solutions
    3. Patterns to reuse or avoid
    4. Recommendations for similar future work

    Save to docs/solutions/ in the workspace.
    """

    inject_mail_context(base, "developer", data.project_id)
  end
end
