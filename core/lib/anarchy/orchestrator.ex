defmodule Anarchy.Orchestrator do
  @moduledoc """
  Polls the task database and dispatches repository copies to agent workers.
  """

  use GenServer
  require Logger
  import Bitwise, only: [<<<: 2]

  alias Anarchy.{AgentRunner, Config, StatusDashboard, Tracker, Workspace}
  alias Anarchy.Schemas.Task, as: TaskSchema
  alias Anarchy.Workers.CELoopWorker

  # Roles that go through the CE loop (WorkflowEngine via Oban)
  @ce_loop_roles ~w(developer senior_developer qa_engineer)

  @continuation_retry_delay_ms 1_000
  @failure_retry_base_ms 10_000
  # Slightly above the dashboard render interval so "checking now…" can render.
  @poll_transition_render_delay_ms 20
  @empty_codex_totals %{
    input_tokens: 0,
    output_tokens: 0,
    total_tokens: 0,
    seconds_running: 0
  }

  defmodule State do
    @moduledoc """
    Runtime state for the orchestrator polling loop.
    """

    defstruct [
      :poll_interval_ms,
      :max_concurrent_agents,
      :next_poll_due_at_ms,
      :poll_check_in_progress,
      :tick_timer_ref,
      :tick_token,
      running: %{},
      completed: MapSet.new(),
      claimed: MapSet.new(),
      retry_attempts: %{},
      codex_totals: nil,
      codex_rate_limits: nil
    ]
  end

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    name = Keyword.get(opts, :name, __MODULE__)
    GenServer.start_link(__MODULE__, opts, name: name)
  end

  @impl true
  def init(_opts) do
    now_ms = System.monotonic_time(:millisecond)
    config = Config.settings!()

    state = %State{
      poll_interval_ms: config.polling.interval_ms,
      max_concurrent_agents: config.agent.max_concurrent_agents,
      next_poll_due_at_ms: now_ms,
      poll_check_in_progress: false,
      tick_timer_ref: nil,
      tick_token: nil,
      codex_totals: @empty_codex_totals,
      codex_rate_limits: nil
    }

    run_terminal_workspace_cleanup()
    state = schedule_tick(state, 0)

    {:ok, state}
  end

  @impl true
  def handle_info({:tick, tick_token}, %{tick_token: tick_token} = state)
      when is_reference(tick_token) do
    state = refresh_runtime_config(state)

    state = %{
      state
      | poll_check_in_progress: true,
        next_poll_due_at_ms: nil,
        tick_timer_ref: nil,
        tick_token: nil
    }

    notify_dashboard()
    :ok = schedule_poll_cycle_start()
    {:noreply, state}
  end

  def handle_info({:tick, _tick_token}, state), do: {:noreply, state}

  def handle_info(:tick, state) do
    state = refresh_runtime_config(state)

    state = %{
      state
      | poll_check_in_progress: true,
        next_poll_due_at_ms: nil,
        tick_timer_ref: nil,
        tick_token: nil
    }

    notify_dashboard()
    :ok = schedule_poll_cycle_start()
    {:noreply, state}
  end

  def handle_info(:run_poll_cycle, state) do
    state = refresh_runtime_config(state)
    state = maybe_dispatch(state)
    state = schedule_tick(state, state.poll_interval_ms)
    state = %{state | poll_check_in_progress: false}

    notify_dashboard()
    {:noreply, state}
  end

  def handle_info(
        {:DOWN, ref, :process, _pid, reason},
        %{running: running} = state
      ) do
    case find_task_id_for_ref(running, ref) do
      nil ->
        {:noreply, state}

      task_id ->
        {running_entry, state} = pop_running_entry(state, task_id)
        state = record_session_completion_totals(state, running_entry)
        session_id = running_entry_session_id(running_entry)

        state =
          case reason do
            :normal ->
              Logger.info("Agent task completed for task_id=#{task_id} session_id=#{session_id}; scheduling active-state continuation check")

              state
              |> complete_task(task_id)
              |> schedule_task_retry(task_id, 1, %{
                title: running_entry.title,
                delay_type: :continuation
              })

            _ ->
              Logger.warning("Agent task exited for task_id=#{task_id} session_id=#{session_id} reason=#{inspect(reason)}; scheduling retry")

              next_attempt = next_retry_attempt_from_running(running_entry)

              schedule_task_retry(state, task_id, next_attempt, %{
                title: running_entry.title,
                error: "agent exited: #{inspect(reason)}"
              })
          end

        Logger.info("Agent task finished for task_id=#{task_id} session_id=#{session_id} reason=#{inspect(reason)}")

        notify_dashboard()
        {:noreply, state}
    end
  end

  def handle_info(
        {:codex_worker_update, task_id, %{event: _, timestamp: _} = update},
        %{running: running} = state
      ) do
    case Map.get(running, task_id) do
      nil ->
        {:noreply, state}

      running_entry ->
        {updated_running_entry, token_delta} = integrate_codex_update(running_entry, update)

        state =
          state
          |> apply_codex_token_delta(token_delta)
          |> apply_codex_rate_limits(update)

        notify_dashboard()
        {:noreply, %{state | running: Map.put(running, task_id, updated_running_entry)}}
    end
  end

  def handle_info({:codex_worker_update, _task_id, _update}, state), do: {:noreply, state}

  def handle_info({:retry_task, task_id, retry_token}, state) do
    result =
      case pop_retry_attempt_state(state, task_id, retry_token) do
        {:ok, attempt, metadata, state} -> handle_retry_task(state, task_id, attempt, metadata)
        :missing -> {:noreply, state}
      end

    notify_dashboard()
    result
  end

  def handle_info({:retry_task, _task_id}, state), do: {:noreply, state}

  def handle_info(msg, state) do
    Logger.debug("Orchestrator ignored message: #{inspect(msg)}")
    {:noreply, state}
  end

  defp maybe_dispatch(%State{} = state) do
    state = reconcile_running_tasks(state)

    with :ok <- Config.validate!(),
         {:ok, tasks} <- Tracker.fetch_candidate_tasks(),
         true <- available_slots(state) > 0 do
      choose_tasks(tasks, state)
    else
      {:error, :missing_tracker_kind} ->
        Logger.error("Tracker kind missing in WORKFLOW.md")

        state

      {:error, {:unsupported_tracker_kind, kind}} ->
        Logger.error("Unsupported tracker kind in WORKFLOW.md: #{inspect(kind)}")

        state

      {:error, {:invalid_workflow_config, message}} ->
        Logger.error("Invalid WORKFLOW.md config: #{message}")
        state

      {:error, {:missing_workflow_file, path, reason}} ->
        Logger.error("Missing WORKFLOW.md at #{path}: #{inspect(reason)}")
        state

      {:error, :workflow_front_matter_not_a_map} ->
        Logger.error("Failed to parse WORKFLOW.md: workflow front matter must decode to a map")
        state

      {:error, {:workflow_parse_error, reason}} ->
        Logger.error("Failed to parse WORKFLOW.md: #{inspect(reason)}")
        state

      {:error, reason} ->
        Logger.error("Failed to fetch tasks: #{inspect(reason)}")
        state

      false ->
        state
    end
  end

  defp reconcile_running_tasks(%State{} = state) do
    state = reconcile_stalled_running_tasks(state)
    running_ids = Map.keys(state.running)

    if running_ids == [] do
      state
    else
      case Tracker.fetch_task_states_by_ids(running_ids) do
        {:ok, task_states_map} ->
          task_states_map
          |> reconcile_running_task_states(
            state,
            active_state_set(),
            terminal_state_set()
          )
          |> reconcile_missing_running_task_ids(running_ids, task_states_map)

        {:error, reason} ->
          Logger.debug("Failed to refresh running task states: #{inspect(reason)}; keeping active workers")

          state
      end
    end
  end

  @doc false
  @spec reconcile_task_states_for_test(map(), term()) :: term()
  def reconcile_task_states_for_test(task_states_map, %State{} = state) when is_map(task_states_map) do
    reconcile_running_task_states(task_states_map, state, active_state_set(), terminal_state_set())
  end

  def reconcile_task_states_for_test(task_states_map, state) when is_map(task_states_map) do
    reconcile_running_task_states(task_states_map, state, active_state_set(), terminal_state_set())
  end

  @doc false
  @spec should_dispatch_task_for_test(TaskSchema.t(), term()) :: boolean()
  def should_dispatch_task_for_test(%TaskSchema{} = task, %State{} = state) do
    should_dispatch_task?(task, state, active_state_set(), terminal_state_set())
  end

  @doc false
  @spec revalidate_task_for_dispatch_for_test(TaskSchema.t(), ([String.t()] -> term())) ::
          {:ok, TaskSchema.t()} | {:skip, TaskSchema.t() | :missing} | {:error, term()}
  def revalidate_task_for_dispatch_for_test(%TaskSchema{} = task, task_fetcher)
      when is_function(task_fetcher, 1) do
    revalidate_task_for_dispatch(task, task_fetcher, terminal_state_set())
  end

  @doc false
  @spec sort_tasks_for_dispatch_for_test([TaskSchema.t()]) :: [TaskSchema.t()]
  def sort_tasks_for_dispatch_for_test(tasks) when is_list(tasks) do
    sort_tasks_for_dispatch(tasks)
  end

  # task_states_map is %{task_id => status_atom} from fetch_task_states_by_ids
  defp reconcile_running_task_states(task_states_map, state, active_states, terminal_states)
       when is_map(task_states_map) do
    Enum.reduce(task_states_map, state, fn {task_id, status}, state_acc ->
      reconcile_task_state(task_id, status, state_acc, active_states, terminal_states)
    end)
  end

  defp reconcile_task_state(task_id, status, state, active_states, terminal_states) when is_atom(status) do
    status_str = Atom.to_string(status)

    cond do
      terminal_task_state?(status_str, terminal_states) ->
        Logger.info("Task moved to terminal state: task_id=#{task_id} status=#{status}; stopping active agent")

        terminate_running_task(state, task_id, true)

      active_task_state?(status_str, active_states) ->
        refresh_running_task_status(state, task_id, status)

      true ->
        Logger.info("Task moved to non-active state: task_id=#{task_id} status=#{status}; stopping active agent")

        terminate_running_task(state, task_id, false)
    end
  end

  defp reconcile_task_state(_task_id, _status, state, _active_states, _terminal_states), do: state

  defp reconcile_missing_running_task_ids(%State{} = state, requested_task_ids, task_states_map)
       when is_list(requested_task_ids) and is_map(task_states_map) do
    visible_task_ids = MapSet.new(Map.keys(task_states_map))

    Enum.reduce(requested_task_ids, state, fn task_id, state_acc ->
      if MapSet.member?(visible_task_ids, task_id) do
        state_acc
      else
        log_missing_running_task(state_acc, task_id)
        terminate_running_task(state_acc, task_id, false)
      end
    end)
  end

  defp reconcile_missing_running_task_ids(state, _requested_task_ids, _task_states_map), do: state

  defp log_missing_running_task(%State{} = state, task_id) when is_binary(task_id) do
    case Map.get(state.running, task_id) do
      %{title: title} ->
        Logger.info("Task no longer visible during running-state refresh: task_id=#{task_id} title=#{title}; stopping active agent")

      _ ->
        Logger.info("Task no longer visible during running-state refresh: task_id=#{task_id}; stopping active agent")
    end
  end

  defp log_missing_running_task(_state, _task_id), do: :ok

  defp refresh_running_task_status(%State{} = state, task_id, status) do
    case Map.get(state.running, task_id) do
      %{task: task} = running_entry when not is_nil(task) ->
        updated_task = %{task | status: status}
        %{state | running: Map.put(state.running, task_id, %{running_entry | task: updated_task})}

      _ ->
        state
    end
  end

  defp terminate_running_task(%State{} = state, task_id, cleanup_workspace) do
    case Map.get(state.running, task_id) do
      nil ->
        release_task_claim(state, task_id)

      %{pid: pid, ref: ref, title: title} = running_entry ->
        state = record_session_completion_totals(state, running_entry)

        if cleanup_workspace do
          cleanup_task_workspace(title)
        end

        if is_pid(pid) do
          terminate_task_process(pid)
        end

        if is_reference(ref) do
          Process.demonitor(ref, [:flush])
        end

        %{
          state
          | running: Map.delete(state.running, task_id),
            claimed: MapSet.delete(state.claimed, task_id),
            retry_attempts: Map.delete(state.retry_attempts, task_id)
        }

      _ ->
        release_task_claim(state, task_id)
    end
  end

  defp reconcile_stalled_running_tasks(%State{} = state) do
    timeout_ms = Config.settings!().codex.stall_timeout_ms

    cond do
      timeout_ms <= 0 ->
        state

      map_size(state.running) == 0 ->
        state

      true ->
        now = DateTime.utc_now()

        Enum.reduce(state.running, state, fn {task_id, running_entry}, state_acc ->
          restart_stalled_task(state_acc, task_id, running_entry, now, timeout_ms)
        end)
    end
  end

  defp restart_stalled_task(state, task_id, running_entry, now, timeout_ms) do
    elapsed_ms = stall_elapsed_ms(running_entry, now)

    if is_integer(elapsed_ms) and elapsed_ms > timeout_ms do
      title = Map.get(running_entry, :title, task_id)
      session_id = running_entry_session_id(running_entry)

      Logger.warning("Task stalled: task_id=#{task_id} title=#{title} session_id=#{session_id} elapsed_ms=#{elapsed_ms}; restarting with backoff")

      next_attempt = next_retry_attempt_from_running(running_entry)

      state
      |> terminate_running_task(task_id, false)
      |> schedule_task_retry(task_id, next_attempt, %{
        title: title,
        error: "stalled for #{elapsed_ms}ms without agent activity"
      })
    else
      state
    end
  end

  defp stall_elapsed_ms(running_entry, now) do
    running_entry
    |> last_activity_timestamp()
    |> case do
      %DateTime{} = timestamp ->
        max(0, DateTime.diff(now, timestamp, :millisecond))

      _ ->
        nil
    end
  end

  defp last_activity_timestamp(running_entry) when is_map(running_entry) do
    Map.get(running_entry, :last_codex_timestamp) || Map.get(running_entry, :started_at)
  end

  defp last_activity_timestamp(_running_entry), do: nil

  defp terminate_task_process(pid) when is_pid(pid) do
    case Task.Supervisor.terminate_child(Anarchy.TaskSupervisor, pid) do
      :ok ->
        :ok

      {:error, :not_found} ->
        Process.exit(pid, :shutdown)
    end
  end

  defp terminate_task_process(_pid), do: :ok

  defp choose_tasks(tasks, state) do
    active_states = active_state_set()
    terminal_states = terminal_state_set()

    tasks
    |> sort_tasks_for_dispatch()
    |> Enum.reduce(state, fn task, state_acc ->
      if should_dispatch_task?(task, state_acc, active_states, terminal_states) do
        dispatch_task(state_acc, task)
      else
        state_acc
      end
    end)
  end

  defp sort_tasks_for_dispatch(tasks) when is_list(tasks) do
    Enum.sort_by(tasks, fn
      %TaskSchema{} = task ->
        {priority_rank(task.priority), task_created_at_sort_key(task), task.id || ""}

      _ ->
        {priority_rank(nil), task_created_at_sort_key(nil), ""}
    end)
  end

  defp priority_rank(priority) when is_integer(priority) and priority in 1..4, do: priority
  defp priority_rank(_priority), do: 5

  defp task_created_at_sort_key(%TaskSchema{inserted_at: %NaiveDateTime{} = inserted_at}) do
    inserted_at |> NaiveDateTime.to_erl() |> :calendar.datetime_to_gregorian_seconds()
  end

  defp task_created_at_sort_key(%TaskSchema{}), do: 9_223_372_036_854_775_807
  defp task_created_at_sort_key(_task), do: 9_223_372_036_854_775_807

  defp should_dispatch_task?(
         %TaskSchema{} = task,
         %State{running: running, claimed: claimed} = state,
         active_states,
         terminal_states
       ) do
    candidate_task?(task, active_states, terminal_states) and
      !task_blocked_by_dependencies?(task) and
      !MapSet.member?(claimed, task.id) and
      !Map.has_key?(running, task.id) and
      available_slots(state) > 0 and
      state_slots_available?(task, running)
  end

  defp should_dispatch_task?(_task, _state, _active_states, _terminal_states), do: false

  defp state_slots_available?(%TaskSchema{status: status}, running) when is_map(running) do
    status_str = status_to_string(status)
    limit = Config.max_concurrent_agents_for_state(status_str)
    used = running_task_count_for_status(running, status_str)
    limit > used
  end

  defp state_slots_available?(_task, _running), do: false

  defp running_task_count_for_status(running, status_str) when is_map(running) do
    normalized = normalize_task_state(status_str)

    Enum.count(running, fn
      {_id, %{task: %TaskSchema{status: s}}} ->
        normalize_task_state(status_to_string(s)) == normalized

      _ ->
        false
    end)
  end

  defp candidate_task?(
         %TaskSchema{
           id: id,
           title: title,
           status: status
         } = _task,
         active_states,
         terminal_states
       )
       when not is_nil(id) and is_binary(title) and is_atom(status) do
    status_str = status_to_string(status)

    active_task_state?(status_str, active_states) and
      !terminal_task_state?(status_str, terminal_states)
  end

  defp candidate_task?(_task, _active_states, _terminal_states), do: false

  defp task_blocked_by_dependencies?(%TaskSchema{depends_on: deps})
       when is_list(deps) and deps != [] do
    # Dependencies are checked at the tracker/query level; this is a safety fallback
    false
  end

  defp task_blocked_by_dependencies?(_task), do: false

  defp terminal_task_state?(status_str, terminal_states) when is_binary(status_str) do
    MapSet.member?(terminal_states, normalize_task_state(status_str))
  end

  defp terminal_task_state?(_status_str, _terminal_states), do: false

  defp active_task_state?(status_str, active_states) when is_binary(status_str) do
    MapSet.member?(active_states, normalize_task_state(status_str))
  end

  defp normalize_task_state(state_name) when is_binary(state_name) do
    String.downcase(String.trim(state_name))
  end

  defp status_to_string(status) when is_atom(status), do: Atom.to_string(status)
  defp status_to_string(status) when is_binary(status), do: status

  defp terminal_state_set do
    Config.settings!().tracker.terminal_states
    |> Enum.map(&normalize_task_state/1)
    |> Enum.filter(&(&1 != ""))
    |> MapSet.new()
  end

  defp active_state_set do
    Config.settings!().tracker.active_states
    |> Enum.map(&normalize_task_state/1)
    |> Enum.filter(&(&1 != ""))
    |> MapSet.new()
  end

  defp dispatch_task(%State{} = state, task, attempt \\ nil) do
    case revalidate_task_for_dispatch(task, &Tracker.fetch_task_states_by_ids/1, terminal_state_set()) do
      {:ok, %TaskSchema{} = refreshed_task} ->
        case dispatch_mode(refreshed_task.role) do
          :ce_loop -> dispatch_ce_loop(state, refreshed_task)
          :direct -> do_dispatch_task(state, refreshed_task, attempt)
        end

      {:skip, :missing} ->
        Logger.info("Skipping dispatch; task no longer active or visible: #{task_context(task)}")
        state

      {:skip, %TaskSchema{} = refreshed_task} ->
        Logger.info("Skipping stale dispatch after task refresh: #{task_context(refreshed_task)} status=#{inspect(refreshed_task.status)}")

        state

      {:error, reason} ->
        Logger.warning("Skipping dispatch; task refresh failed for #{task_context(task)}: #{inspect(reason)}")
        state
    end
  end

  # CE loop roles go through WorkflowEngine via Oban. Review roles are dispatched
  # by the WorkflowEngine itself, not directly by the Orchestrator.
  defp dispatch_mode(role) when is_binary(role) and role in @ce_loop_roles, do: :ce_loop
  defp dispatch_mode(_role), do: :direct

  defp dispatch_ce_loop(%State{} = state, task) do
    case CELoopWorker.enqueue(task.id, task.project_id) do
      {:ok, _job} ->
        Logger.info("Enqueued CE loop for #{task_context(task)}")
        # Add to claimed to prevent re-dispatch on next poll tick.
        # CE loop tasks are NOT in the running map — Oban owns their lifecycle.
        %{state | claimed: MapSet.put(state.claimed, task.id)}

      {:error, reason} ->
        Logger.warning("Failed to enqueue CE loop for #{task_context(task)}: #{inspect(reason)}")
        state
    end
  end

  defp do_dispatch_task(%State{} = state, task, attempt) do
    recipient = self()

    case Task.Supervisor.start_child(Anarchy.TaskSupervisor, fn ->
           AgentRunner.run(task, recipient, attempt: attempt)
         end) do
      {:ok, pid} ->
        ref = Process.monitor(pid)

        Logger.info("Dispatching task to agent: #{task_context(task)} pid=#{inspect(pid)} attempt=#{inspect(attempt)}")

        running =
          Map.put(state.running, task.id, %{
            pid: pid,
            ref: ref,
            title: task.title,
            task: task,
            session_id: nil,
            last_codex_message: nil,
            last_codex_timestamp: nil,
            last_codex_event: nil,
            codex_app_server_pid: nil,
            codex_input_tokens: 0,
            codex_output_tokens: 0,
            codex_total_tokens: 0,
            codex_last_reported_input_tokens: 0,
            codex_last_reported_output_tokens: 0,
            codex_last_reported_total_tokens: 0,
            turn_count: 0,
            retry_attempt: normalize_retry_attempt(attempt),
            started_at: DateTime.utc_now()
          })

        %{
          state
          | running: running,
            claimed: MapSet.put(state.claimed, task.id),
            retry_attempts: Map.delete(state.retry_attempts, task.id)
        }

      {:error, reason} ->
        Logger.error("Unable to spawn agent for #{task_context(task)}: #{inspect(reason)}")
        next_attempt = if is_integer(attempt), do: attempt + 1, else: nil

        schedule_task_retry(state, task.id, next_attempt, %{
          title: task.title,
          error: "failed to spawn agent: #{inspect(reason)}"
        })
    end
  end

  # revalidate_task_for_dispatch checks the current status of a task via the fetcher.
  # The fetcher returns {:ok, %{id => status_atom}}, so we reconstruct a refreshed task.
  defp revalidate_task_for_dispatch(%TaskSchema{id: task_id} = task, task_fetcher, terminal_states)
       when not is_nil(task_id) and is_function(task_fetcher, 1) do
    case task_fetcher.([task_id]) do
      {:ok, task_states_map} when is_map(task_states_map) ->
        case Map.get(task_states_map, task_id) do
          nil ->
            {:skip, :missing}

          status ->
            refreshed_task = %{task | status: status}

            if retry_candidate_task?(refreshed_task, terminal_states) do
              {:ok, refreshed_task}
            else
              {:skip, refreshed_task}
            end
        end

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp revalidate_task_for_dispatch(task, _task_fetcher, _terminal_states), do: {:ok, task}

  defp complete_task(%State{} = state, task_id) do
    %{
      state
      | completed: MapSet.put(state.completed, task_id),
        retry_attempts: Map.delete(state.retry_attempts, task_id)
    }
  end

  defp schedule_task_retry(%State{} = state, task_id, attempt, metadata)
       when is_binary(task_id) and is_map(metadata) do
    previous_retry = Map.get(state.retry_attempts, task_id, %{attempt: 0})
    next_attempt = if is_integer(attempt), do: attempt, else: previous_retry.attempt + 1
    delay_ms = retry_delay(next_attempt, metadata)
    old_timer = Map.get(previous_retry, :timer_ref)
    retry_token = make_ref()
    due_at_ms = System.monotonic_time(:millisecond) + delay_ms
    title = pick_retry_title(task_id, previous_retry, metadata)
    error = pick_retry_error(previous_retry, metadata)

    if is_reference(old_timer) do
      Process.cancel_timer(old_timer)
    end

    timer_ref = Process.send_after(self(), {:retry_task, task_id, retry_token}, delay_ms)

    error_suffix = if is_binary(error), do: " error=#{error}", else: ""

    Logger.warning("Retrying task_id=#{task_id} title=#{title} in #{delay_ms}ms (attempt #{next_attempt})#{error_suffix}")

    %{
      state
      | retry_attempts:
          Map.put(state.retry_attempts, task_id, %{
            attempt: next_attempt,
            timer_ref: timer_ref,
            retry_token: retry_token,
            due_at_ms: due_at_ms,
            title: title,
            error: error
          })
    }
  end

  defp pop_retry_attempt_state(%State{} = state, task_id, retry_token) when is_reference(retry_token) do
    case Map.get(state.retry_attempts, task_id) do
      %{attempt: attempt, retry_token: ^retry_token} = retry_entry ->
        metadata = %{
          title: Map.get(retry_entry, :title),
          error: Map.get(retry_entry, :error)
        }

        {:ok, attempt, metadata, %{state | retry_attempts: Map.delete(state.retry_attempts, task_id)}}

      _ ->
        :missing
    end
  end

  defp handle_retry_task(%State{} = state, task_id, attempt, metadata) do
    case Tracker.fetch_candidate_tasks() do
      {:ok, tasks} ->
        tasks
        |> find_task_by_id(task_id)
        |> handle_retry_task_lookup(state, task_id, attempt, metadata)

      {:error, reason} ->
        Logger.warning("Retry poll failed for task_id=#{task_id} title=#{metadata[:title] || task_id}: #{inspect(reason)}")

        {:noreply,
         schedule_task_retry(
           state,
           task_id,
           attempt + 1,
           Map.merge(metadata, %{error: "retry poll failed: #{inspect(reason)}"})
         )}
    end
  end

  defp handle_retry_task_lookup(%TaskSchema{} = task, state, task_id, attempt, metadata) do
    terminal_states = terminal_state_set()
    status_str = status_to_string(task.status)

    cond do
      terminal_task_state?(status_str, terminal_states) ->
        Logger.info("Task status is terminal: task_id=#{task_id} title=#{task.title} status=#{task.status}; removing associated workspace")

        cleanup_task_workspace(task.title)
        {:noreply, release_task_claim(state, task_id)}

      retry_candidate_task?(task, terminal_states) ->
        handle_active_retry(state, task, attempt, metadata)

      true ->
        Logger.debug("Task left active states, removing claim task_id=#{task_id} title=#{task.title}")

        {:noreply, release_task_claim(state, task_id)}
    end
  end

  defp handle_retry_task_lookup(nil, state, task_id, _attempt, _metadata) do
    Logger.debug("Task no longer visible, removing claim task_id=#{task_id}")
    {:noreply, release_task_claim(state, task_id)}
  end

  defp cleanup_task_workspace(identifier) when is_binary(identifier) do
    Workspace.remove_issue_workspaces(identifier)
  end

  defp cleanup_task_workspace(_identifier), do: :ok

  defp run_terminal_workspace_cleanup do
    case Tracker.fetch_tasks_by_states(Config.settings!().tracker.terminal_states) do
      {:ok, tasks} ->
        tasks
        |> Enum.each(fn
          %TaskSchema{title: title} when is_binary(title) ->
            cleanup_task_workspace(title)

          _ ->
            :ok
        end)

      {:error, reason} ->
        Logger.warning("Skipping startup terminal workspace cleanup; failed to fetch terminal tasks: #{inspect(reason)}")
    end
  end

  defp notify_dashboard do
    StatusDashboard.notify_update()
  end

  defp handle_active_retry(state, task, attempt, metadata) do
    if retry_candidate_task?(task, terminal_state_set()) and
         dispatch_slots_available?(task, state) do
      {:noreply, dispatch_task(state, task, attempt)}
    else
      Logger.debug("No available slots for retrying #{task_context(task)}; retrying again")

      {:noreply,
       schedule_task_retry(
         state,
         task.id,
         attempt + 1,
         Map.merge(metadata, %{
           title: task.title,
           error: "no available orchestrator slots"
         })
       )}
    end
  end

  defp release_task_claim(%State{} = state, task_id) do
    %{state | claimed: MapSet.delete(state.claimed, task_id)}
  end

  defp retry_delay(attempt, metadata) when is_integer(attempt) and attempt > 0 and is_map(metadata) do
    if metadata[:delay_type] == :continuation and attempt == 1 do
      @continuation_retry_delay_ms
    else
      failure_retry_delay(attempt)
    end
  end

  defp failure_retry_delay(attempt) do
    max_delay_power = min(attempt - 1, 10)
    min(@failure_retry_base_ms * (1 <<< max_delay_power), Config.settings!().agent.max_retry_backoff_ms)
  end

  defp normalize_retry_attempt(attempt) when is_integer(attempt) and attempt > 0, do: attempt
  defp normalize_retry_attempt(_attempt), do: 0

  defp next_retry_attempt_from_running(running_entry) do
    case Map.get(running_entry, :retry_attempt) do
      attempt when is_integer(attempt) and attempt > 0 -> attempt + 1
      _ -> nil
    end
  end

  defp pick_retry_title(task_id, previous_retry, metadata) do
    metadata[:title] || Map.get(previous_retry, :title) || task_id
  end

  defp pick_retry_error(previous_retry, metadata) do
    metadata[:error] || Map.get(previous_retry, :error)
  end

  defp find_task_by_id(tasks, task_id) when is_binary(task_id) do
    Enum.find(tasks, fn
      %TaskSchema{id: ^task_id} ->
        true

      _ ->
        false
    end)
  end

  defp find_task_id_for_ref(running, ref) do
    running
    |> Enum.find_value(fn {task_id, %{ref: running_ref}} ->
      if running_ref == ref, do: task_id
    end)
  end

  defp running_entry_session_id(%{session_id: session_id}) when is_binary(session_id),
    do: session_id

  defp running_entry_session_id(_running_entry), do: "n/a"

  defp task_context(%TaskSchema{id: task_id, title: title}) do
    "task_id=#{task_id} title=#{title}"
  end

  defp available_slots(%State{} = state) do
    max(
      (state.max_concurrent_agents || Config.settings!().agent.max_concurrent_agents) -
        map_size(state.running),
      0
    )
  end

  @spec request_refresh() :: map() | :unavailable
  def request_refresh do
    request_refresh(__MODULE__)
  end

  @spec request_refresh(GenServer.server()) :: map() | :unavailable
  def request_refresh(server) do
    if Process.whereis(server) do
      GenServer.call(server, :request_refresh)
    else
      :unavailable
    end
  end

  @spec snapshot() :: map() | :timeout | :unavailable
  def snapshot, do: snapshot(__MODULE__, 15_000)

  @spec snapshot(GenServer.server(), timeout()) :: map() | :timeout | :unavailable
  def snapshot(server, timeout) do
    if Process.whereis(server) do
      try do
        GenServer.call(server, :snapshot, timeout)
      catch
        :exit, {:timeout, _} -> :timeout
        :exit, _ -> :unavailable
      end
    else
      :unavailable
    end
  end

  @impl true
  def handle_call(:snapshot, _from, state) do
    state = refresh_runtime_config(state)
    now = DateTime.utc_now()
    now_ms = System.monotonic_time(:millisecond)

    running =
      state.running
      |> Enum.map(fn {task_id, metadata} ->
        %{
          task_id: task_id,
          title: metadata.title,
          status: metadata.task.status,
          session_id: metadata.session_id,
          codex_app_server_pid: metadata.codex_app_server_pid,
          codex_input_tokens: metadata.codex_input_tokens,
          codex_output_tokens: metadata.codex_output_tokens,
          codex_total_tokens: metadata.codex_total_tokens,
          turn_count: Map.get(metadata, :turn_count, 0),
          started_at: metadata.started_at,
          last_codex_timestamp: metadata.last_codex_timestamp,
          last_codex_message: metadata.last_codex_message,
          last_codex_event: metadata.last_codex_event,
          runtime_seconds: running_seconds(metadata.started_at, now)
        }
      end)

    retrying =
      state.retry_attempts
      |> Enum.map(fn {task_id, %{attempt: attempt, due_at_ms: due_at_ms} = retry} ->
        %{
          task_id: task_id,
          attempt: attempt,
          due_in_ms: max(0, due_at_ms - now_ms),
          title: Map.get(retry, :title),
          error: Map.get(retry, :error)
        }
      end)

    {:reply,
     %{
       running: running,
       retrying: retrying,
       codex_totals: state.codex_totals,
       rate_limits: Map.get(state, :codex_rate_limits),
       polling: %{
         checking?: state.poll_check_in_progress == true,
         next_poll_in_ms: next_poll_in_ms(state.next_poll_due_at_ms, now_ms),
         poll_interval_ms: state.poll_interval_ms
       }
     }, state}
  end

  def handle_call(:request_refresh, _from, state) do
    now_ms = System.monotonic_time(:millisecond)
    already_due? = is_integer(state.next_poll_due_at_ms) and state.next_poll_due_at_ms <= now_ms
    coalesced = state.poll_check_in_progress == true or already_due?
    state = if coalesced, do: state, else: schedule_tick(state, 0)

    {:reply,
     %{
       queued: true,
       coalesced: coalesced,
       requested_at: DateTime.utc_now(),
       operations: ["poll", "reconcile"]
     }, state}
  end

  defp integrate_codex_update(running_entry, %{event: event, timestamp: timestamp} = update) do
    token_delta = extract_token_delta(running_entry, update)
    codex_input_tokens = Map.get(running_entry, :codex_input_tokens, 0)
    codex_output_tokens = Map.get(running_entry, :codex_output_tokens, 0)
    codex_total_tokens = Map.get(running_entry, :codex_total_tokens, 0)
    codex_app_server_pid = Map.get(running_entry, :codex_app_server_pid)
    last_reported_input = Map.get(running_entry, :codex_last_reported_input_tokens, 0)
    last_reported_output = Map.get(running_entry, :codex_last_reported_output_tokens, 0)
    last_reported_total = Map.get(running_entry, :codex_last_reported_total_tokens, 0)
    turn_count = Map.get(running_entry, :turn_count, 0)

    {
      Map.merge(running_entry, %{
        last_codex_timestamp: timestamp,
        last_codex_message: summarize_codex_update(update),
        session_id: session_id_for_update(running_entry.session_id, update),
        last_codex_event: event,
        codex_app_server_pid: codex_app_server_pid_for_update(codex_app_server_pid, update),
        codex_input_tokens: codex_input_tokens + token_delta.input_tokens,
        codex_output_tokens: codex_output_tokens + token_delta.output_tokens,
        codex_total_tokens: codex_total_tokens + token_delta.total_tokens,
        codex_last_reported_input_tokens: max(last_reported_input, token_delta.input_reported),
        codex_last_reported_output_tokens: max(last_reported_output, token_delta.output_reported),
        codex_last_reported_total_tokens: max(last_reported_total, token_delta.total_reported),
        turn_count: turn_count_for_update(turn_count, running_entry.session_id, update)
      }),
      token_delta
    }
  end

  defp codex_app_server_pid_for_update(_existing, %{codex_app_server_pid: pid})
       when is_binary(pid),
       do: pid

  defp codex_app_server_pid_for_update(_existing, %{codex_app_server_pid: pid})
       when is_integer(pid),
       do: Integer.to_string(pid)

  defp codex_app_server_pid_for_update(_existing, %{codex_app_server_pid: pid}) when is_list(pid),
    do: to_string(pid)

  defp codex_app_server_pid_for_update(existing, _update), do: existing

  defp session_id_for_update(_existing, %{session_id: session_id}) when is_binary(session_id),
    do: session_id

  defp session_id_for_update(existing, _update), do: existing

  defp turn_count_for_update(existing_count, existing_session_id, %{
         event: :session_started,
         session_id: session_id
       })
       when is_integer(existing_count) and is_binary(session_id) do
    if session_id == existing_session_id do
      existing_count
    else
      existing_count + 1
    end
  end

  defp turn_count_for_update(existing_count, _existing_session_id, _update)
       when is_integer(existing_count),
       do: existing_count

  defp turn_count_for_update(_existing_count, _existing_session_id, _update), do: 0

  defp summarize_codex_update(update) do
    %{
      event: update[:event],
      message: update[:payload] || update[:raw],
      timestamp: update[:timestamp]
    }
  end

  defp schedule_tick(%State{} = state, delay_ms) when is_integer(delay_ms) and delay_ms >= 0 do
    if is_reference(state.tick_timer_ref) do
      Process.cancel_timer(state.tick_timer_ref)
    end

    tick_token = make_ref()
    timer_ref = Process.send_after(self(), {:tick, tick_token}, delay_ms)

    %{
      state
      | tick_timer_ref: timer_ref,
        tick_token: tick_token,
        next_poll_due_at_ms: System.monotonic_time(:millisecond) + delay_ms
    }
  end

  defp schedule_poll_cycle_start do
    :timer.send_after(@poll_transition_render_delay_ms, self(), :run_poll_cycle)
    :ok
  end

  defp next_poll_in_ms(nil, _now_ms), do: nil

  defp next_poll_in_ms(next_poll_due_at_ms, now_ms) when is_integer(next_poll_due_at_ms) do
    max(0, next_poll_due_at_ms - now_ms)
  end

  defp pop_running_entry(state, task_id) do
    {Map.get(state.running, task_id), %{state | running: Map.delete(state.running, task_id)}}
  end

  defp record_session_completion_totals(state, running_entry) when is_map(running_entry) do
    runtime_seconds = running_seconds(running_entry.started_at, DateTime.utc_now())

    codex_totals =
      apply_token_delta(
        state.codex_totals,
        %{
          input_tokens: 0,
          output_tokens: 0,
          total_tokens: 0,
          seconds_running: runtime_seconds
        }
      )

    %{state | codex_totals: codex_totals}
  end

  defp record_session_completion_totals(state, _running_entry), do: state

  defp refresh_runtime_config(%State{} = state) do
    config = Config.settings!()

    %{
      state
      | poll_interval_ms: config.polling.interval_ms,
        max_concurrent_agents: config.agent.max_concurrent_agents
    }
  end

  defp retry_candidate_task?(%TaskSchema{} = task, terminal_states) do
    candidate_task?(task, active_state_set(), terminal_states) and
      !task_blocked_by_dependencies?(task)
  end

  defp dispatch_slots_available?(%TaskSchema{} = task, %State{} = state) do
    available_slots(state) > 0 and state_slots_available?(task, state.running)
  end

  defp apply_codex_token_delta(
         %{codex_totals: codex_totals} = state,
         %{input_tokens: input, output_tokens: output, total_tokens: total} = token_delta
       )
       when is_integer(input) and is_integer(output) and is_integer(total) do
    %{state | codex_totals: apply_token_delta(codex_totals, token_delta)}
  end

  defp apply_codex_token_delta(state, _token_delta), do: state

  defp apply_codex_rate_limits(%State{} = state, update) when is_map(update) do
    case extract_rate_limits(update) do
      %{} = rate_limits ->
        %{state | codex_rate_limits: rate_limits}

      _ ->
        state
    end
  end

  defp apply_codex_rate_limits(state, _update), do: state

  defp apply_token_delta(codex_totals, token_delta) do
    input_tokens = Map.get(codex_totals, :input_tokens, 0) + token_delta.input_tokens
    output_tokens = Map.get(codex_totals, :output_tokens, 0) + token_delta.output_tokens
    total_tokens = Map.get(codex_totals, :total_tokens, 0) + token_delta.total_tokens

    seconds_running =
      Map.get(codex_totals, :seconds_running, 0) + Map.get(token_delta, :seconds_running, 0)

    %{
      input_tokens: max(0, input_tokens),
      output_tokens: max(0, output_tokens),
      total_tokens: max(0, total_tokens),
      seconds_running: max(0, seconds_running)
    }
  end

  defp extract_token_delta(running_entry, %{event: _, timestamp: _} = update) do
    running_entry = running_entry || %{}
    usage = extract_token_usage(update)

    {
      compute_token_delta(
        running_entry,
        :input,
        usage,
        :codex_last_reported_input_tokens
      ),
      compute_token_delta(
        running_entry,
        :output,
        usage,
        :codex_last_reported_output_tokens
      ),
      compute_token_delta(
        running_entry,
        :total,
        usage,
        :codex_last_reported_total_tokens
      )
    }
    |> Tuple.to_list()
    |> then(fn [input, output, total] ->
      %{
        input_tokens: input.delta,
        output_tokens: output.delta,
        total_tokens: total.delta,
        input_reported: input.reported,
        output_reported: output.reported,
        total_reported: total.reported
      }
    end)
  end

  defp compute_token_delta(running_entry, token_key, usage, reported_key) do
    next_total = get_token_usage(usage, token_key)
    prev_reported = Map.get(running_entry, reported_key, 0)

    delta =
      if is_integer(next_total) and next_total >= prev_reported do
        next_total - prev_reported
      else
        0
      end

    %{
      delta: max(delta, 0),
      reported: if(is_integer(next_total), do: next_total, else: prev_reported)
    }
  end

  defp extract_token_usage(update) do
    payloads = [
      update[:usage],
      Map.get(update, "usage"),
      Map.get(update, :usage),
      update[:payload],
      Map.get(update, "payload"),
      update
    ]

    Enum.find_value(payloads, &absolute_token_usage_from_payload/1) ||
      Enum.find_value(payloads, &turn_completed_usage_from_payload/1) ||
      %{}
  end

  defp extract_rate_limits(update) do
    rate_limits_from_payload(update[:rate_limits]) ||
      rate_limits_from_payload(Map.get(update, "rate_limits")) ||
      rate_limits_from_payload(Map.get(update, :rate_limits)) ||
      rate_limits_from_payload(update[:payload]) ||
      rate_limits_from_payload(Map.get(update, "payload")) ||
      rate_limits_from_payload(update)
  end

  defp absolute_token_usage_from_payload(payload) when is_map(payload) do
    absolute_paths = [
      ["params", "msg", "payload", "info", "total_token_usage"],
      [:params, :msg, :payload, :info, :total_token_usage],
      ["params", "msg", "info", "total_token_usage"],
      [:params, :msg, :info, :total_token_usage],
      ["params", "tokenUsage", "total"],
      [:params, :tokenUsage, :total],
      ["tokenUsage", "total"],
      [:tokenUsage, :total]
    ]

    explicit_map_at_paths(payload, absolute_paths)
  end

  defp absolute_token_usage_from_payload(_payload), do: nil

  defp turn_completed_usage_from_payload(payload) when is_map(payload) do
    method = Map.get(payload, "method") || Map.get(payload, :method)

    if method in ["turn/completed", :turn_completed] do
      direct =
        Map.get(payload, "usage") ||
          Map.get(payload, :usage) ||
          map_at_path(payload, ["params", "usage"]) ||
          map_at_path(payload, [:params, :usage])

      if is_map(direct) and integer_token_map?(direct), do: direct
    end
  end

  defp turn_completed_usage_from_payload(_payload), do: nil

  defp rate_limits_from_payload(payload) when is_map(payload) do
    direct = Map.get(payload, "rate_limits") || Map.get(payload, :rate_limits)

    cond do
      rate_limits_map?(direct) ->
        direct

      rate_limits_map?(payload) ->
        payload

      true ->
        rate_limit_payloads(payload)
    end
  end

  defp rate_limits_from_payload(payload) when is_list(payload) do
    rate_limit_payloads(payload)
  end

  defp rate_limits_from_payload(_payload), do: nil

  defp rate_limit_payloads(payload) when is_map(payload) do
    Map.values(payload)
    |> Enum.reduce_while(nil, fn
      value, nil ->
        case rate_limits_from_payload(value) do
          nil -> {:cont, nil}
          rate_limits -> {:halt, rate_limits}
        end

      _value, result ->
        {:halt, result}
    end)
  end

  defp rate_limit_payloads(payload) when is_list(payload) do
    payload
    |> Enum.reduce_while(nil, fn
      value, nil ->
        case rate_limits_from_payload(value) do
          nil -> {:cont, nil}
          rate_limits -> {:halt, rate_limits}
        end

      _value, result ->
        {:halt, result}
    end)
  end

  defp rate_limits_map?(payload) when is_map(payload) do
    limit_id =
      Map.get(payload, "limit_id") ||
        Map.get(payload, :limit_id) ||
        Map.get(payload, "limit_name") ||
        Map.get(payload, :limit_name)

    has_buckets =
      Enum.any?(
        ["primary", :primary, "secondary", :secondary, "credits", :credits],
        &Map.has_key?(payload, &1)
      )

    !is_nil(limit_id) and has_buckets
  end

  defp rate_limits_map?(_payload), do: false

  defp explicit_map_at_paths(payload, paths) when is_map(payload) and is_list(paths) do
    Enum.find_value(paths, fn path ->
      value = map_at_path(payload, path)

      if is_map(value) and integer_token_map?(value), do: value
    end)
  end

  defp explicit_map_at_paths(_payload, _paths), do: nil

  defp map_at_path(payload, path) when is_map(payload) and is_list(path) do
    Enum.reduce_while(path, payload, fn key, acc ->
      if is_map(acc) and Map.has_key?(acc, key) do
        {:cont, Map.get(acc, key)}
      else
        {:halt, nil}
      end
    end)
  end

  defp map_at_path(_payload, _path), do: nil

  defp integer_token_map?(payload) do
    token_fields = [
      :input_tokens,
      :output_tokens,
      :total_tokens,
      :prompt_tokens,
      :completion_tokens,
      :inputTokens,
      :outputTokens,
      :totalTokens,
      :promptTokens,
      :completionTokens,
      "input_tokens",
      "output_tokens",
      "total_tokens",
      "prompt_tokens",
      "completion_tokens",
      "inputTokens",
      "outputTokens",
      "totalTokens",
      "promptTokens",
      "completionTokens"
    ]

    token_fields
    |> Enum.any?(fn field ->
      value = payload_get(payload, field)
      !is_nil(integer_like(value))
    end)
  end

  defp get_token_usage(usage, :input),
    do:
      payload_get(usage, [
        "input_tokens",
        "prompt_tokens",
        :input_tokens,
        :prompt_tokens,
        :input,
        "promptTokens",
        :promptTokens,
        "inputTokens",
        :inputTokens
      ])

  defp get_token_usage(usage, :output),
    do:
      payload_get(usage, [
        "output_tokens",
        "completion_tokens",
        :output_tokens,
        :completion_tokens,
        :output,
        :completion,
        "outputTokens",
        :outputTokens,
        "completionTokens",
        :completionTokens
      ])

  defp get_token_usage(usage, :total),
    do:
      payload_get(usage, [
        "total_tokens",
        "total",
        :total_tokens,
        :total,
        "totalTokens",
        :totalTokens
      ])

  defp payload_get(payload, fields) when is_list(fields) do
    Enum.find_value(fields, fn field -> map_integer_value(payload, field) end)
  end

  defp payload_get(payload, field), do: map_integer_value(payload, field)

  defp map_integer_value(payload, field) do
    if is_map(payload) do
      value = Map.get(payload, field)
      integer_like(value)
    else
      nil
    end
  end

  defp running_seconds(%DateTime{} = started_at, %DateTime{} = now) do
    max(0, DateTime.diff(now, started_at, :second))
  end

  defp running_seconds(_started_at, _now), do: 0

  defp integer_like(value) when is_integer(value) and value >= 0, do: value

  defp integer_like(value) when is_binary(value) do
    case Integer.parse(String.trim(value)) do
      {num, _} when num >= 0 -> num
      _ -> nil
    end
  end

  defp integer_like(_value), do: nil
end
