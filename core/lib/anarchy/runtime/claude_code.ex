defmodule Anarchy.Runtime.ClaudeCode do
  @moduledoc """
  Claude Code CLI runtime.

  Two execution modes:
  - `run_once/1`: One-shot execution, blocks until completion, returns `{:ok, text}`.
  - `start_interactive/1` + `send_message/2`: Persistent session for multi-turn chat.
  """
  use GenServer

  @behaviour Anarchy.Runtime.AgentProtocol

  require Logger

  defstruct [:port, :session_id, :workspace_path, :on_message, :buffer]

  # --- One-shot execution (plain function, no GenServer) ---

  @doc """
  One-shot Claude Code execution. Blocks the calling process until the Port exits.
  Returns `{:ok, text}` with the assistant's final output, or `{:error, reason}`.
  """
  @impl Anarchy.Runtime.AgentProtocol
  @spec run_once(keyword()) :: {:ok, String.t()} | {:error, term()}
  def run_once(opts) do
    :ok = validate_run_once_opts(opts)
    args = build_run_once_args(opts)
    executable = find_claude_executable()
    port_opts = [:binary, :exit_status, :use_stdio, args: args]
    port_opts = if opts[:workspace_path], do: [{:cd, opts[:workspace_path]} | port_opts], else: port_opts

    port = Port.open({:spawn_executable, executable}, port_opts)
    timeout = opts[:timeout] || 3_600_000
    collect_output(port, "", "", timeout)
  end

  # --- Interactive session (GenServer-based) ---

  @impl Anarchy.Runtime.AgentProtocol
  @doc "Start interactive Claude session (no -p). Returns {:ok, session_id, pid}."
  @spec start_interactive(keyword()) :: {:ok, String.t(), pid()}
  def start_interactive(opts) do
    opts = Map.new(opts) |> Map.put(:mode, :interactive)
    {:ok, pid} = GenServer.start_link(__MODULE__, opts)
    session_id = GenServer.call(pid, :get_session_id)
    {:ok, session_id, pid}
  end

  @impl Anarchy.Runtime.AgentProtocol
  @doc "Resume a previous interactive session via --resume."
  @spec resume_session(String.t(), keyword()) :: {:ok, pid()}
  def resume_session(session_id, opts \\ []) do
    {:ok, pid} = GenServer.start_link(__MODULE__, Map.new(opts) |> Map.put(:resume, session_id))
    {:ok, pid}
  end

  @impl Anarchy.Runtime.AgentProtocol
  @doc "Send a message to an interactive session's stdin."
  @spec send_message(pid(), String.t()) :: :ok
  def send_message(pid, message) do
    GenServer.cast(pid, {:send_message, message})
  end

  @impl Anarchy.Runtime.AgentProtocol
  @doc "Stop an interactive session."
  @spec stop_session(pid()) :: :ok
  def stop_session(pid) do
    GenServer.stop(pid, :normal)
  end

  # --- GenServer callbacks ---

  @impl GenServer
  def init(%{resume: session_id} = opts) do
    args = ["--resume", session_id, "--output-format", "stream-json"]
    args = if skip_permissions?(), do: ["--dangerously-skip-permissions" | args], else: args
    port = open_port(args, opts[:workspace_path])

    {:ok,
     %__MODULE__{
       port: port,
       session_id: session_id,
       on_message: opts[:on_message],
       buffer: ""
     }}
  end

  def init(%{mode: :interactive} = opts) do
    session_id = opts[:session_id] || generate_session_id()
    args = build_interactive_args(opts)
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
  def handle_cast({:send_message, message}, state) do
    Port.command(state.port, message <> "\n")
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

  @impl GenServer
  def handle_info({port, {:exit_status, status}}, %{port: port} = state) do
    Logger.info("Claude Code exited with status #{status} for session #{state.session_id}")
    {:stop, :normal, %{state | port: nil}}
  end

  @impl GenServer
  def terminate(_reason, %{port: port} = _state) when is_port(port) do
    kill_port_process(port)
    :ok
  end

  @impl GenServer
  def terminate(_reason, _state), do: :ok

  # --- Private: run_once helpers ---

  defp build_run_once_args(opts) do
    base = ["-p", opts[:prompt] || "", "--output-format", "stream-json"]

    base =
      if skip_permissions?(),
        do: ["--dangerously-skip-permissions" | base],
        else: base

    base
    |> maybe_add("--model", opts[:model])
    |> maybe_add("--system-prompt", opts[:system_prompt])
    |> maybe_add("--max-budget-usd", opts[:budget])
  end

  # remaining = unparsed buffer fragment (NOT full history)
  # acc_text = accumulated result text
  defp collect_output(port, remaining, acc_text, timeout) do
    receive do
      {^port, {:data, data}} ->
        {messages, new_remaining} = parse_stream_json(remaining <> data)
        new_text = extract_result_text(messages)
        collect_output(port, new_remaining, acc_text <> new_text, timeout)

      {^port, {:exit_status, 0}} ->
        # Parse any remaining buffer
        {messages, _} = parse_stream_json(remaining)
        final_text = acc_text <> extract_result_text(messages)
        if final_text == "", do: {:error, :empty_output}, else: {:ok, final_text}

      {^port, {:exit_status, status}} ->
        {:error, {:exit_status, status}}
    after
      timeout ->
        kill_port_process(port)
        {:error, :timeout}
    end
  end

  defp extract_result_text(messages) do
    Enum.flat_map(messages, fn
      %{"type" => "result", "result" => text} when is_binary(text) -> [text]
      _ -> []
    end)
    |> Enum.join("")
  end

  # --- Private: interactive helpers ---

  defp build_interactive_args(opts) do
    base = ["--output-format", "stream-json"]

    base =
      if skip_permissions?(),
        do: ["--dangerously-skip-permissions" | base],
        else: base

    base
    |> maybe_add("--model", opts[:model])
    |> maybe_add("--system-prompt", opts[:system_prompt])
  end

  # --- Private: shared ---

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

  defp maybe_add(args, _flag, nil), do: args
  defp maybe_add(args, flag, value), do: args ++ [flag, to_string(value)]

  defp skip_permissions? do
    try do
      Anarchy.Config.settings!().claude_code.skip_permissions
    rescue
      error ->
        Logger.warning("Failed to read skip_permissions config, defaulting to false: #{inspect(error)}")
        false
    end
  end

  defp kill_port_process(port) do
    os_pid =
      case :erlang.port_info(port, :os_pid) do
        {:os_pid, pid} -> pid
        _ -> nil
      end

    Port.close(port)

    if os_pid do
      case :os.type() do
        {:win32, _} -> System.cmd("taskkill", ["/F", "/T", "/PID", to_string(os_pid)], stderr_to_stdout: true)
        _ -> System.cmd("kill", ["-9", to_string(os_pid)], stderr_to_stdout: true)
      end
    end
  rescue
    error ->
      Logger.warning("Failed to kill port process: #{inspect(error)}")
  end

  defp validate_run_once_opts(opts) do
    unless Keyword.has_key?(opts, :prompt) and is_binary(opts[:prompt]) and opts[:prompt] != "" do
      raise ArgumentError, ":prompt is required and must be a non-empty string"
    end

    for {key, value} <- opts, key in [:prompt, :model, :system_prompt, :workspace_path], value != nil do
      str = to_string(value)

      if String.contains?(str, "\0") do
        raise ArgumentError, "#{key} must not contain null bytes"
      end

      if key in [:model, :workspace_path] and String.starts_with?(str, "-") do
        raise ArgumentError, "#{key} must not start with a dash"
      end
    end

    :ok
  end

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
