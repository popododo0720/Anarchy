defmodule AnarchyWeb.ArchitectChatLive do
  @moduledoc """
  Architect Chat UI — the owner talks to the Architect agent to create designs.
  Messages are streamed in real-time via PubSub.
  """

  use Phoenix.LiveView

  alias Anarchy.{Projects, RoleLoader}

  @impl true
  def mount(%{"project_id" => project_id}, _session, socket) do
    project = Projects.get_project!(project_id)

    if connected?(socket) do
      Phoenix.PubSub.subscribe(Anarchy.PubSub, "architect:#{project_id}")
    end

    {:ok,
     assign(socket,
       page_title: "Architect Chat — #{project.name}",
       project: project,
       messages: [],
       input: "",
       streaming: false,
       session_id: nil,
       worker_pid: nil
     )}
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
              <div class="chat-content"><span class="typing-indicator">...</span></div>
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

    # Start architect agent in background
    project = socket.assigns.project
    caller = self()

    pid =
      spawn(fn ->
        try do
          response = run_architect(project, message)
          send(caller, {:architect_response, response})
        rescue
          error ->
            send(caller, {:architect_response, "Error: #{Exception.message(error)}"})
        end
      end)

    {:noreply,
     assign(socket,
       messages: messages,
       input: "",
       streaming: true,
       worker_pid: pid
     )}
  end

  def handle_event("send_message", _params, socket), do: {:noreply, socket}

  def handle_event("save_as_design", _params, socket) do
    messages = socket.assigns.messages
    project = socket.assigns.project

    # Combine all assistant messages as design content
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
    {:noreply, assign(socket, messages: [], streaming: false)}
  end

  @impl true
  def handle_info({:architect_response, response}, socket) do
    messages = socket.assigns.messages ++ [%{role: "architect", content: response}]
    {:noreply, assign(socket, messages: messages, streaming: false, worker_pid: nil)}
  end

  def handle_info({:architect_stream, chunk}, socket) do
    # For streaming partial responses
    {:noreply, assign(socket, streaming_content: (socket.assigns[:streaming_content] || "") <> chunk)}
  end

  def handle_info(_msg, socket), do: {:noreply, socket}

  # --- Private ---

  defp run_architect(project, message) do
    case RoleLoader.load(:architect) do
      {:ok, _system_prompt} ->
        # In production, this would call Claude Code with the system prompt + prompt below.
        # For now, return a placeholder indicating the agent would respond.
        "I've analyzed your request for project '#{project.name}'.\n\n" <>
          "Based on your message, here's my initial design thinking:\n\n" <>
          "#{message}\n\n" <>
          "To proceed, I would need Claude Code CLI available to provide detailed architectural guidance. " <>
          "You can save this conversation as a design document when ready."

      {:error, _reason} ->
        "Architect role prompt not available. Please ensure priv/agency-agents/engineering/architect.md exists."
    end
  end

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
