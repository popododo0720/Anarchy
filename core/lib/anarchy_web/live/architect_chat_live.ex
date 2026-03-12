defmodule AnarchyWeb.ArchitectChatLive do
  @moduledoc """
  Architect Chat UI — the owner talks to the Architect agent to create designs.
  Uses ClaudeCode interactive sessions with real-time streaming via PubSub.
  """

  use Phoenix.LiveView

  alias Anarchy.{Projects, RoleLoader, SessionManager}
  alias Anarchy.Runtime.ClaudeCode
  require Logger

  @impl true
  def mount(%{"project_id" => project_id}, _session, socket) do
    project = Projects.get_project!(project_id)

    socket =
      assign(socket,
        page_title: "Architect Chat — #{project.name}",
        project: project,
        messages: [],
        input: "",
        streaming: false,
        stream_buffer: "",
        session_id: nil,
        claude_pid: nil
      )

    if connected?(socket) do
      socket = start_architect_session(socket, project)
      {:ok, socket}
    else
      {:ok, socket}
    end
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="dashboard">
      <header class="dashboard-header">
        <h1>
          <a href="/projects" class="text-muted">Projects</a> /
          <a href={"/projects/#{@project.id}"} class="text-muted"><%= @project.name %></a> /
          Architect Chat
        </h1>
      </header>

      <div class="chat-container">
        <div class="chat-messages" id="chat-messages" phx-hook="ScrollBottom">
          <%= for msg <- @messages do %>
            <div class={"chat-message chat-#{msg.role}"}>
              <div class="chat-role"><%= msg.role %></div>
              <div class="chat-content"><pre><%= msg.content %></pre></div>
            </div>
          <% end %>

          <%= if @streaming do %>
            <div class="chat-message chat-assistant streaming">
              <div class="chat-role">architect</div>
              <div class="chat-content">
                <%= if @stream_buffer != "" do %>
                  <pre><%= @stream_buffer %></pre>
                <% else %>
                  <span class="typing-indicator">...</span>
                <% end %>
              </div>
            </div>
          <% end %>

          <%= if @messages == [] do %>
            <div class="empty-state">
              <p>Start a conversation with the Architect agent.</p>
              <p class="text-muted">Describe what you want to build and the Architect will help create a design document.</p>
            </div>
          <% end %>
        </div>

        <div class="chat-input-area">
          <form phx-submit="send_message" class="chat-form">
            <textarea
              name="message"
              rows="3"
              class="input chat-input"
              placeholder="Describe what you want to build..."
              disabled={@streaming}
            ><%= @input %></textarea>
            <button type="submit" class="btn btn-primary" disabled={@streaming}>
              <%= if @streaming, do: "Thinking...", else: "Send" %>
            </button>
          </form>

          <div class="chat-actions">
            <button phx-click="save_as_design" class="btn btn-sm" disabled={@messages == []}>
              Save as Design
            </button>
            <button phx-click="clear_chat" class="btn btn-sm">Clear</button>
          </div>
        </div>
      </div>
    </div>
    """
  end

  @impl true
  def handle_event("send_message", %{"message" => message}, socket) when message != "" do
    messages = socket.assigns.messages ++ [%{role: "user", content: message}]

    socket =
      if socket.assigns.claude_pid && Process.alive?(socket.assigns.claude_pid) do
        # Send to existing interactive session
        ClaudeCode.send_message(socket.assigns.claude_pid, message)
        assign(socket, messages: messages, input: "", streaming: true, stream_buffer: "")
      else
        # Session died or not started — unsubscribe old topic, start fresh
        old_topic = socket.assigns[:session_id] && "agent:#{socket.assigns.session_id}"
        if old_topic, do: Phoenix.PubSub.unsubscribe(Anarchy.PubSub, old_topic)

        socket = assign(socket, messages: messages, input: "", stream_buffer: "")
        socket = start_architect_session(socket, socket.assigns.project)

        if socket.assigns.claude_pid && Process.alive?(socket.assigns.claude_pid) do
          ClaudeCode.send_message(socket.assigns.claude_pid, message)
          assign(socket, streaming: true)
        else
          # Session failed to start — don't leave streaming stuck
          assign(socket, streaming: false)
        end
      end

    {:noreply, socket}
  end

  def handle_event("send_message", _params, socket), do: {:noreply, socket}

  def handle_event("save_as_design", _params, socket) do
    messages = socket.assigns.messages
    project = socket.assigns.project

    content =
      messages
      |> Enum.filter(&(&1.role == "architect"))
      |> Enum.map(& &1.content)
      |> Enum.join("\n\n---\n\n")

    title = extract_title(messages) || "Design from Architect Chat"

    case Projects.create_design(%{
           project_id: project.id,
           title: title,
           content_md: content,
           status: "draft"
         }) do
      {:ok, design} ->
        {:noreply,
         socket
         |> put_flash(:info, "Design saved: #{design.title}")
         |> redirect(to: "/designs/#{design.id}")}

      {:error, _changeset} ->
        {:noreply, put_flash(socket, :error, "Failed to save design")}
    end
  end

  def handle_event("clear_chat", _params, socket) do
    {:noreply, assign(socket, messages: [], streaming: false, stream_buffer: "")}
  end

  # Streaming: "result" type message completes the response (specific clause FIRST)
  @impl true
  def handle_info({:agent_output, _sid, %{"type" => "result", "result" => text}}, socket) when is_binary(text) do
    final_text = socket.assigns.stream_buffer <> text
    messages = socket.assigns.messages ++ [%{role: "architect", content: final_text}]
    {:noreply, assign(socket, messages: messages, streaming: false, stream_buffer: "")}
  end

  def handle_info({:agent_output, _sid, %{"type" => "result"}}, socket) do
    # Result without text — use accumulated buffer
    final_text = socket.assigns.stream_buffer

    if final_text != "" do
      messages = socket.assigns.messages ++ [%{role: "architect", content: final_text}]
      {:noreply, assign(socket, messages: messages, streaming: false, stream_buffer: "")}
    else
      {:noreply, assign(socket, streaming: false)}
    end
  end

  # Streaming: accumulate text chunks from assistant messages
  def handle_info({:agent_output, _sid, msg}, socket) do
    case extract_text_chunk(msg) do
      nil -> {:noreply, socket}
      chunk -> {:noreply, assign(socket, stream_buffer: socket.assigns.stream_buffer <> chunk)}
    end
  end

  def handle_info(_msg, socket), do: {:noreply, socket}

  @impl true
  def terminate(_reason, socket) do
    # Stop the Claude CLI process to prevent orphans
    if socket.assigns[:claude_pid] && Process.alive?(socket.assigns.claude_pid) do
      ClaudeCode.stop_session(socket.assigns.claude_pid)
    end

    if socket.assigns[:session_id] do
      SessionManager.complete_session(socket.assigns.session_id)
    end

    :ok
  end

  # --- Private ---

  # Max concurrent interactive sessions per project
  @max_interactive_sessions 3

  defp start_architect_session(socket, project) do
    active_count =
      try do
        Projects.list_sessions(project.id)
        |> Enum.count(fn s -> s.status == "active" and s.agent_type == "claude_code" end)
      rescue
        _ -> 0
      end

    if active_count >= @max_interactive_sessions do
      Logger.warning("Interactive session limit reached for project_id=#{project.id}")
      assign(socket, session_id: nil, claude_pid: nil)
    else
      do_start_architect_session(socket, project)
    end
  end

  defp do_start_architect_session(socket, project) do
    system_prompt =
      case RoleLoader.load(:architect) do
        {:ok, content} -> content
        {:error, _} -> nil
      end

    try do
      {:ok, session_id, pid} =
        ClaudeCode.start_interactive(
          model: RoleLoader.model_for(:architect),
          system_prompt: system_prompt,
          workspace_path: nil
        )

      Phoenix.PubSub.subscribe(Anarchy.PubSub, "agent:#{session_id}")

      SessionManager.create_session(%{
        project_id: project.id,
        agent_type: "claude_code",
        session_id: session_id,
        status: "active"
      })

      assign(socket, session_id: session_id, claude_pid: pid)
    rescue
      error ->
        Logger.warning("Failed to start architect session: #{Exception.message(error)}")
        assign(socket, session_id: nil, claude_pid: nil)
    end
  end

  defp extract_text_chunk(%{"type" => "assistant", "message" => %{"content" => content}}) when is_list(content) do
    Enum.flat_map(content, fn
      %{"type" => "text", "text" => text} when is_binary(text) -> [text]
      _ -> []
    end)
    |> Enum.join("")
    |> case do
      "" -> nil
      text -> text
    end
  end

  defp extract_text_chunk(%{"type" => "content_block_delta", "delta" => %{"text" => text}}) when is_binary(text) do
    text
  end

  defp extract_text_chunk(_msg), do: nil

  defp extract_title(messages) do
    case Enum.find(messages, &(&1.role == "user")) do
      %{content: content} ->
        content
        |> String.split("\n")
        |> List.first()
        |> String.slice(0, 100)

      nil ->
        nil
    end
  end
end
