defmodule Anarchy.ErrorHandler do
  @moduledoc """
  Centralized error handling, classification, and recovery strategies.

  Classifies errors into categories and determines appropriate recovery:
  - :transient — retry with backoff (network, rate limit)
  - :permanent — fail fast, no retry (bad config, missing resource)
  - :escalate — needs human intervention (merge conflict, auth failure)
  """

  require Logger

  @type error_class :: :transient | :permanent | :escalate
  @type recovery :: :retry | :fail | :escalate | :skip

  @spec classify(term()) :: {error_class(), recovery(), String.t()}
  def classify(error) do
    case error do
      # Rate limiting
      {:rate_limited, _details} ->
        {:transient, :retry, "Rate limited — will retry with backoff"}

      # Network errors
      {:network_error, _reason} ->
        {:transient, :retry, "Network error — will retry"}

      %Req.TransportError{} ->
        {:transient, :retry, "HTTP transport error — will retry"}

      # Session errors
      {:session_expired, _id} ->
        {:transient, :retry, "Session expired — will create new session with resume context"}

      # Agent runtime errors
      {:claude_code_exited, :normal} ->
        {:permanent, :skip, "Agent completed normally"}

      {:claude_code_exited, reason} ->
        {:transient, :retry, "Claude Code exited unexpectedly: #{inspect(reason)}"}

      {:codex_exited, reason} ->
        {:transient, :retry, "Codex exited unexpectedly: #{inspect(reason)}"}

      # Database errors
      %Postgrex.Error{postgres: %{code: :unique_violation}} ->
        {:permanent, :fail, "Unique constraint violation"}

      %Postgrex.Error{postgres: %{code: :foreign_key_violation}} ->
        {:permanent, :fail, "Foreign key violation — referenced record doesn't exist"}

      %Postgrex.Error{} ->
        {:transient, :retry, "Database error — will retry"}

      # Configuration errors
      {:invalid_workflow_config, msg} ->
        {:permanent, :fail, "Invalid configuration: #{msg}"}

      {:missing_workflow_file, _path, _reason} ->
        {:permanent, :fail, "Missing WORKFLOW.md"}

      # Merge conflicts
      {:conflict, _details} ->
        {:escalate, :escalate, "Merge conflict — needs manual resolution"}

      # Task state errors
      {:task_not_found, _id} ->
        {:permanent, :fail, "Task not found"}

      {:design_not_confirmed, _status} ->
        {:permanent, :fail, "Design must be confirmed before decomposition"}

      # Timeout
      :timeout ->
        {:transient, :retry, "Operation timed out — will retry"}

      :ce_loop_timeout ->
        {:permanent, :fail, "CE loop exceeded maximum time"}

      # Cancelled
      :cancelled ->
        {:permanent, :skip, "Operation cancelled by user"}

      # Generic
      %RuntimeError{message: msg} ->
        {:transient, :retry, "Runtime error: #{msg}"}

      other ->
        {:transient, :retry, "Unknown error: #{inspect(other, limit: 100)}"}
    end
  end

  @spec handle(term(), keyword()) :: :ok | {:error, term()}
  def handle(error, opts \\ []) do
    {class, recovery, message} = classify(error)
    context = Keyword.get(opts, :context, "unknown")
    task_id = Keyword.get(opts, :task_id)

    Logger.warning("Error in #{context}: [#{class}] #{message} task_id=#{task_id || "n/a"}")

    case recovery do
      :retry ->
        Logger.info("Will retry after backoff for #{context}")
        :ok

      :fail ->
        Logger.error("Permanent failure in #{context}: #{message}")
        {:error, error}

      :escalate ->
        Logger.error("Escalation needed for #{context}: #{message}")
        broadcast_escalation(task_id, message)
        {:error, {:needs_escalation, message}}

      :skip ->
        Logger.info("Skipping error in #{context}: #{message}")
        :ok
    end
  end

  @spec retriable?(term()) :: boolean()
  def retriable?(error) do
    {_class, recovery, _msg} = classify(error)
    recovery == :retry
  end

  @spec backoff_ms(integer()) :: integer()
  def backoff_ms(attempt) when is_integer(attempt) and attempt >= 0 do
    base = 10_000
    max = 300_000
    delay = base * :math.pow(2, attempt) |> trunc()
    min(delay, max)
  end

  # --- Private ---

  defp broadcast_escalation(task_id, message) do
    Phoenix.PubSub.broadcast(
      Anarchy.PubSub,
      "escalations",
      {:escalation, %{task_id: task_id, message: message, timestamp: DateTime.utc_now()}}
    )
  rescue
    _ -> :ok
  end
end
