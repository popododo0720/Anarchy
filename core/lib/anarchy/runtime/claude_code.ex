defmodule Anarchy.Runtime.ClaudeCode do
  @moduledoc "Claude Code CLI runtime worker."
  @behaviour Anarchy.Runtime.AgentProtocol
  use GenServer

  require Logger

  defstruct [:port, :session_id, :workspace_path, :on_message, :buffer]

  # --- AgentProtocol callbacks ---

  @impl Anarchy.Runtime.AgentProtocol
  def start_session(opts) do
    {:ok, pid} = GenServer.start_link(__MODULE__, opts)
    session_id = GenServer.call(pid, :get_session_id)
    {:ok, session_id, pid}
  end

  @impl Anarchy.Runtime.AgentProtocol
  def resume_session(session_id) do
    {:ok, pid} = GenServer.start_link(__MODULE__, %{resume: session_id})
    {:ok, pid}
  end

  @impl Anarchy.Runtime.AgentProtocol
  def send_prompt(pid, prompt) do
    GenServer.cast(pid, {:send_prompt, prompt})
  end

  @impl Anarchy.Runtime.AgentProtocol
  def stop_session(pid) do
    GenServer.stop(pid, :normal)
  end

  # --- GenServer callbacks ---

  @impl GenServer
  def init(%{resume: session_id} = opts) do
    args = ["--resume", session_id, "--output-format", "stream-json"]
    port = open_port(args, opts[:workspace_path])

    {:ok,
     %__MODULE__{
       port: port,
       session_id: session_id,
       on_message: opts[:on_message],
       buffer: ""
     }}
  end

  def init(opts) do
    session_id = opts[:session_id] || generate_session_id()
    args = build_args(opts)
    port = open_port(args, opts[:workspace_path])

    {:ok,
     %__MODULE__{
       port: port,
       session_id: session_id,
       workspace_path: opts[:workspace_path],
       on_message: opts[:on_message],
       buffer: ""
     }}
  end

  @impl GenServer
  def handle_call(:get_session_id, _from, state) do
    {:reply, state.session_id, state}
  end

  @impl GenServer
  def handle_cast({:send_prompt, prompt}, state) do
    Port.command(state.port, prompt <> "\n")
    {:noreply, state}
  end

  @impl GenServer
  def handle_info({port, {:data, data}}, %{port: port} = state) do
    {messages, remaining} = parse_stream_json(state.buffer <> data)

    for msg <- messages do
      if state.on_message, do: state.on_message.(msg)

      Phoenix.PubSub.broadcast(
        Anarchy.PubSub,
        "agent:#{state.session_id}",
        {:agent_output, state.session_id, msg}
      )
    end

    {:noreply, %{state | buffer: remaining}}
  end

  def handle_info({port, {:exit_status, status}}, %{port: port} = state) do
    Logger.info("Claude Code exited with status #{status} for session #{state.session_id}")
    {:stop, :normal, %{state | port: nil}}
  end

  @impl GenServer
  def terminate(_reason, %{port: port} = _state) when is_port(port) do
    if Port.info(port) != nil, do: Port.close(port)
    :ok
  end

  def terminate(_reason, _state), do: :ok

  # --- Private ---

  defp open_port(args, workspace_path) do
    executable = find_claude_executable()
    port_opts = [:binary, :exit_status, :use_stdio, args: args]
    port_opts = if workspace_path, do: [{:cd, workspace_path} | port_opts], else: port_opts
    Port.open({:spawn_executable, executable}, port_opts)
  end

  defp find_claude_executable do
    System.find_executable("claude") ||
      raise "Claude Code CLI not found. Install: npm install -g @anthropic-ai/claude-code"
  end

  defp build_args(opts) do
    base = ["-p", opts[:prompt] || "", "--output-format", "stream-json"]

    base
    |> maybe_add("--model", opts[:model])
    |> maybe_add("--system-prompt", opts[:system_prompt])
    |> maybe_add("--max-budget-usd", opts[:budget])
  end

  defp maybe_add(args, _flag, nil), do: args
  defp maybe_add(args, flag, value), do: args ++ [flag, to_string(value)]

  defp generate_session_id do
    "anarchy-" <> Base.encode16(:crypto.strong_rand_bytes(8), case: :lower)
  end

  defp parse_stream_json(data) do
    lines = String.split(data, "\n")
    {complete, [remaining]} = Enum.split(lines, -1)

    messages =
      complete
      |> Enum.reject(&(&1 == ""))
      |> Enum.flat_map(fn line ->
        case Jason.decode(line) do
          {:ok, parsed} -> [parsed]
          {:error, _} -> []
        end
      end)

    {messages, remaining}
  end
end
