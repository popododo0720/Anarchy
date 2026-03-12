defmodule Anarchy.AgentRunner do
  @moduledoc """
  Executes a single task in its workspace with the appropriate runtime (Claude Code or Codex).
  """

  require Logger
  alias Anarchy.Codex.AppServer
  alias Anarchy.Runtime.ClaudeCode
  alias Anarchy.Schemas.Task, as: TaskSchema
  alias Anarchy.{Config, PromptBuilder, Tracker, Workspace}

  # Roles that use the Codex (AppServer) runtime
  @codex_roles ~w(plan_reviewer code_reviewer)

  @spec run(map(), pid() | nil, keyword()) :: :ok | no_return()
  def run(task, update_recipient \\ nil, opts \\ []) do
    Logger.info("Starting agent run for #{task_context(task)}")

    case Workspace.create_for_issue(task) do
      {:ok, workspace} ->
        try do
          with :ok <- Workspace.run_before_run_hook(workspace, task),
               :ok <- run_agent_turns(workspace, task, update_recipient, opts) do
            :ok
          else
            {:error, reason} ->
              Logger.error("Agent run failed for #{task_context(task)}: #{inspect(reason)}")
              raise RuntimeError, "Agent run failed for #{task_context(task)}: #{inspect(reason)}"
          end
        after
          Workspace.run_after_run_hook(workspace, task)
        end

      {:error, reason} ->
        Logger.error("Agent run failed for #{task_context(task)}: #{inspect(reason)}")
        raise RuntimeError, "Agent run failed for #{task_context(task)}: #{inspect(reason)}"
    end
  end

  defp run_agent_turns(workspace, task, update_recipient, opts) do
    if use_codex_runtime?(task) do
      run_codex_turns(workspace, task, update_recipient, opts)
    else
      run_claude_code_turns(workspace, task, update_recipient, opts)
    end
  end

  defp use_codex_runtime?(%TaskSchema{role: role}) when is_binary(role) do
    role in @codex_roles
  end

  defp use_codex_runtime?(_task), do: false

  # --- Codex (AppServer) runtime ---

  defp run_codex_turns(workspace, task, update_recipient, opts) do
    max_turns = Keyword.get(opts, :max_turns, Config.settings!().agent.max_turns)
    task_state_fetcher = Keyword.get(opts, :task_state_fetcher, &Tracker.fetch_task_states_by_ids/1)

    with {:ok, session} <- AppServer.start_session(workspace) do
      try do
        do_run_codex_turns(session, workspace, task, update_recipient, opts, task_state_fetcher, 1, max_turns)
      after
        AppServer.stop_session(session)
      end
    end
  end

  defp do_run_codex_turns(app_session, workspace, task, update_recipient, opts, task_state_fetcher, turn_number, max_turns) do
    prompt = build_turn_prompt(task, opts, turn_number, max_turns)

    with {:ok, turn_session} <-
           AppServer.run_turn(
             app_session,
             prompt,
             task,
             on_message: worker_message_handler(update_recipient, task)
           ) do
      Logger.info("Completed agent run for #{task_context(task)} session_id=#{turn_session[:session_id]} workspace=#{workspace} turn=#{turn_number}/#{max_turns}")

      case continue_with_task?(task, task_state_fetcher) do
        {:continue, refreshed_task} when turn_number < max_turns ->
          Logger.info("Continuing agent run for #{task_context(refreshed_task)} after normal turn completion turn=#{turn_number}/#{max_turns}")

          do_run_codex_turns(
            app_session,
            workspace,
            refreshed_task,
            update_recipient,
            opts,
            task_state_fetcher,
            turn_number + 1,
            max_turns
          )

        {:continue, refreshed_task} ->
          Logger.info("Reached agent.max_turns for #{task_context(refreshed_task)} with task still active; returning control to orchestrator")

          :ok

        {:done, _refreshed_task} ->
          :ok

        {:error, reason} ->
          {:error, reason}
      end
    end
  end

  # --- Claude Code runtime ---

  defp run_claude_code_turns(workspace, task, _update_recipient, opts) do
    task_state_fetcher = Keyword.get(opts, :task_state_fetcher, &Tracker.fetch_task_states_by_ids/1)
    prompt = build_turn_prompt(task, opts, 1, Keyword.get(opts, :max_turns, Config.settings!().agent.max_turns))

    case ClaudeCode.run_once(prompt: prompt, workspace_path: workspace) do
      {:ok, _text} ->
        Logger.info("Completed Claude Code agent run for #{task_context(task)} workspace=#{workspace}")

        case continue_with_task?(task, task_state_fetcher) do
          {:continue, _refreshed_task} -> :ok
          {:done, _refreshed_task} -> :ok
          {:error, reason} -> {:error, reason}
        end

      {:error, reason} ->
        {:error, {:claude_code_exited, reason}}
    end
  end

  # --- Common helpers ---

  defp worker_message_handler(recipient, task) do
    fn message ->
      send_worker_update(recipient, task, message)
    end
  end

  defp send_worker_update(recipient, %TaskSchema{id: task_id}, message)
       when is_binary(task_id) and is_pid(recipient) do
    send(recipient, {:codex_worker_update, task_id, message})
    :ok
  end

  defp send_worker_update(_recipient, _task, _message), do: :ok

  defp build_turn_prompt(task, opts, 1, _max_turns), do: PromptBuilder.build_prompt(task, opts)

  defp build_turn_prompt(_task, _opts, turn_number, max_turns) do
    """
    Continuation guidance:

    - The previous turn completed normally, but the task is still in an active state.
    - This is continuation turn ##{turn_number} of #{max_turns} for the current agent run.
    - Resume from the current workspace and workpad state instead of restarting from scratch.
    - The original task instructions and prior turn context are already present in this thread, so do not restate them before acting.
    - Focus on the remaining task work and do not end the turn while the task stays active unless you are truly blocked.
    """
  end

  defp continue_with_task?(%TaskSchema{id: task_id} = task, task_state_fetcher) when not is_nil(task_id) do
    case task_state_fetcher.([task_id]) do
      {:ok, task_states_map} when is_map(task_states_map) ->
        case Map.get(task_states_map, task_id) do
          nil ->
            {:done, task}

          status ->
            if active_task_status?(status) do
              {:continue, %{task | status: status}}
            else
              {:done, %{task | status: status}}
            end
        end

      {:error, reason} ->
        {:error, {:task_state_refresh_failed, reason}}
    end
  end

  defp continue_with_task?(task, _task_state_fetcher), do: {:done, task}

  defp active_task_status?(status) when is_atom(status) do
    status_str = Atom.to_string(status) |> String.trim() |> String.downcase()

    Config.settings!().tracker.active_states
    |> Enum.any?(fn active_state ->
      active_state |> String.trim() |> String.downcase() == status_str
    end)
  end

  defp active_task_status?(_status), do: false

  defp task_context(%{id: task_id, title: title}) do
    "task_id=#{task_id} title=#{title}"
  end
end
