defmodule Anarchy.Runtime.AgentProtocol do
  @moduledoc """
  Common interface for agent runtimes (Claude Code, Codex).

  Two execution modes:
  - One-shot: `run_once/1` — blocks caller, returns `{:ok, text}` or `{:error, reason}`
  - Interactive: `start_interactive/1` + `send_message/2` — persistent session
  """

  @type session_id :: String.t()
  @type opts :: keyword()

  @callback run_once(opts()) :: {:ok, String.t()} | {:error, term()}
  @callback start_interactive(opts()) :: {:ok, session_id(), pid()} | {:error, term()}
  @callback resume_session(session_id()) :: {:ok, pid()} | {:error, term()}
  @callback send_message(pid(), String.t()) :: :ok | {:error, term()}
  @callback stop_session(pid()) :: :ok
end
