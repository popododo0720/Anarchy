defmodule Anarchy.Runtime.AgentProtocol do
  @moduledoc "Common interface for agent runtimes (Claude Code, Codex)."

  @type session_id :: String.t()
  @type opts :: map()

  @callback start_session(opts()) :: {:ok, session_id(), pid()} | {:error, term()}
  @callback resume_session(session_id()) :: {:ok, pid()} | {:error, term()}
  @callback send_prompt(pid(), String.t()) :: :ok | {:error, term()}
  @callback stop_session(pid()) :: :ok
end
