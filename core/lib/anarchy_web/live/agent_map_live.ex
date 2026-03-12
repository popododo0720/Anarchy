defmodule AnarchyWeb.AgentMapLive do
  use Phoenix.LiveView

  alias Anarchy.{AgentMap, AgentMail, Projects}

  @impl true
  def mount(%{"project_id" => project_id}, _session, socket) do
    project = Projects.get_project!(project_id)

    if connected?(socket) do
      Phoenix.PubSub.subscribe(Anarchy.PubSub, "project:#{project_id}")
      Phoenix.PubSub.subscribe(Anarchy.PubSub, "mail:project:#{project_id}")
    end

    tree = AgentMap.build_tree(project_id)

    {:ok,
     assign(socket,
       page_title: "Agent Map — #{project.name}",
       project: project,
       tree: tree,
       selected_agent: nil,
       agent_output: [],
       agent_mails: [],
       inject_input: ""
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
          Agent Map
        </h1>
      </header>

      <div class="agent-map-layout">
        <div class="agent-tree">
          <h3>Agent Hierarchy</h3>

          <div class="tree-section">
            <div class="tree-header">Architect</div>
            <%= for t <- @tree.architect.tasks do %>
              <div class={"tree-node #{if @selected_agent == (t.session && t.session.session_id), do: "selected"}"}>
                <span class={"status-icon status-#{t.status}"}></span>
                <%= if t.session do %>
                  <a href="#" phx-click="select_agent" phx-value-session-id={t.session.session_id}><%= t.title %></a>
                <% else %>
                  <span><%= t.title %></span>
                <% end %>
              </div>
            <% end %>
          </div>

          <%= for pm <- @tree.pms do %>
            <div class="tree-section">
              <div class="tree-header">PM: <%= pm.scope || pm.role %> <span class={"badge badge-#{pm.status}"}><%= pm.status %></span></div>
              <%= for child <- pm.children do %>
                <div class={"tree-node #{if @selected_agent == (child.session && child.session.session_id), do: "selected"}"}>
                  <span class={"status-icon status-#{child.status}"}></span>
                  <%= if child.session do %>
                    <a href="#" phx-click="select_agent" phx-value-session-id={child.session.session_id}><%= child.title %></a>
                  <% else %>
                    <span><%= child.title %></span>
                  <% end %>
                  <%= if child.ce_state do %>
                    <span class="ce-badge"><%= child.ce_state %></span>
                  <% end %>
                </div>
              <% end %>
            </div>
          <% end %>

          <%= if @tree.unassigned_tasks != [] do %>
            <div class="tree-section">
              <div class="tree-header">Unassigned</div>
              <%= for t <- @tree.unassigned_tasks do %>
                <div class="tree-node">
                  <span class={"status-icon status-#{t.status}"}></span>
                  <span><%= t.title %></span>
                </div>
              <% end %>
            </div>
          <% end %>

          <div class="tree-legend">
            <span class="status-icon status-working"></span> Working
            <span class="status-icon status-pending"></span> Pending
            <span class="status-icon status-completed"></span> Done
            <span class="status-icon status-failed"></span> Failed
          </div>
        </div>

        <div class="agent-detail">
          <%= if @selected_agent do %>
            <h3>Agent: <%= @selected_agent %></h3>

            <div class="agent-output-panel">
              <h4>Live Output</h4>
              <div class="log-panel" id="agent-output">
                <%= for line <- @agent_output do %>
                  <div class="log-line"><%= line %></div>
                <% end %>
                <%= if @agent_output == [] do %>
                  <div class="empty-state">No output yet</div>
                <% end %>
              </div>
            </div>

            <div class="agent-mail-panel">
              <h4>Messages</h4>
              <%= for mail <- @agent_mails do %>
                <div class={"mail-item #{if mail.read_at, do: "read", else: "unread"}"}>
                  <strong>[<%= mail.from_agent %>]</strong> <%= mail.subject %>
                  <div class="mail-body"><%= mail.body %></div>
                </div>
              <% end %>
            </div>

            <div class="agent-controls">
              <button phx-click="pause_agent" phx-value-session-id={@selected_agent} class="btn btn-sm">Pause</button>
              <button phx-click="resume_agent" phx-value-session-id={@selected_agent} class="btn btn-sm">Resume</button>
              <form phx-submit="inject_instruction" class="inline-form">
                <input type="hidden" name="session-id" value={@selected_agent} />
                <input type="text" name="message" placeholder="Send instruction..." class="input input-sm" value={@inject_input} />
                <button type="submit" class="btn btn-sm btn-primary">Send</button>
              </form>
            </div>
          <% else %>
            <div class="empty-state">
              <p>Select an agent from the tree to view details.</p>
            </div>
          <% end %>
        </div>
      </div>
    </div>
    """
  end

  @impl true
  def handle_event("select_agent", %{"session-id" => session_id}, socket) do
    # Unsubscribe from previous agent topic to prevent subscription leak
    if socket.assigns[:subscribed_agent_topic] do
      Phoenix.PubSub.unsubscribe(Anarchy.PubSub, socket.assigns.subscribed_agent_topic)
    end

    topic = "agent:#{session_id}"
    Phoenix.PubSub.subscribe(Anarchy.PubSub, topic)

    mails = AgentMail.inbox(session_id, project_id: socket.assigns.project.id)
    {:noreply, assign(socket, selected_agent: session_id, agent_output: [], agent_mails: mails, subscribed_agent_topic: "agent:#{session_id}")}
  end

  def handle_event("pause_agent", %{"session-id" => sid}, socket) do
    Anarchy.SessionManager.pause_session(sid, "manual")
    {:noreply, socket}
  end

  def handle_event("resume_agent", %{"session-id" => sid}, socket) do
    Anarchy.SessionManager.update_session(sid, %{status: "active", paused_at: nil})
    {:noreply, socket}
  end

  def handle_event("inject_instruction", %{"session-id" => sid, "message" => msg}, socket) when msg != "" do
    AgentMail.send(%{
      project_id: socket.assigns.project.id,
      from_agent: "owner",
      to_agent: sid,
      subject: "Owner instruction",
      body: msg,
      type: "assign",
      priority: "urgent"
    })

    {:noreply, assign(socket, inject_input: "")}
  end

  def handle_event("inject_instruction", _params, socket), do: {:noreply, socket}

  @impl true
  def handle_info({:agent_output, session_id, msg}, socket) do
    if socket.assigns.selected_agent == session_id do
      content = if is_map(msg), do: msg["content"] || inspect(msg), else: to_string(msg)
      output = socket.assigns.agent_output ++ [content]
      {:noreply, assign(socket, agent_output: Enum.take(output, -200))}
    else
      {:noreply, socket}
    end
  end

  def handle_info({:task_status_changed, _task_id, _status}, socket) do
    tree = AgentMap.build_tree(socket.assigns.project.id)
    {:noreply, assign(socket, tree: tree)}
  end

  def handle_info({:new_mail, _msg}, socket) do
    if socket.assigns.selected_agent do
      mails = AgentMail.inbox(socket.assigns.selected_agent, project_id: socket.assigns.project.id)
      {:noreply, assign(socket, agent_mails: mails)}
    else
      {:noreply, socket}
    end
  end

  def handle_info(_msg, socket), do: {:noreply, socket}
end
