defmodule AnarchyWeb.AgentMonitorLive do
  @moduledoc """
  Real-time agent monitoring across all projects.
  Shows active sessions, resource usage, and streaming logs.
  """

  use Phoenix.LiveView

  @impl true
  def mount(_params, _session, socket) do
    if connected?(socket) do
      Phoenix.PubSub.subscribe(Anarchy.PubSub, "observability:dashboard")
      :timer.send_interval(2_000, self(), :refresh)
    end

    {:ok,
     assign(socket,
       page_title: "Agent Monitor",
       sessions: load_active_sessions(),
       logs: []
     )}
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="dashboard">
      <header class="dashboard-header">
        <h1>Agent Monitor</h1>
        <nav class="nav-links">
          <a href="/">Dashboard</a>
          <a href="/projects">Projects</a>
          <a href="/agents" class="active">Agents</a>
        </nav>
      </header>

      <div class="metrics-row">
        <div class="metric-card">
          <div class="metric-value"><%= Enum.count(@sessions, &(&1.status == "active")) %></div>
          <div class="metric-label">Active</div>
        </div>
        <div class="metric-card">
          <div class="metric-value"><%= Enum.count(@sessions, &(&1.status == "paused")) %></div>
          <div class="metric-label">Paused</div>
        </div>
        <div class="metric-card">
          <div class="metric-value"><%= length(@sessions) %></div>
          <div class="metric-label">Total</div>
        </div>
      </div>

      <div class="section">
        <h3>Active Sessions</h3>
        <table class="data-table">
          <thead>
            <tr>
              <th>Session ID</th>
              <th>Type</th>
              <th>Status</th>
              <th>Role</th>
              <th>Workspace</th>
              <th>Started</th>
              <th>Last Active</th>
            </tr>
          </thead>
          <tbody>
            <%= for session <- @sessions do %>
              <tr>
                <td class="mono"><%= String.slice(session.session_id, 0, 24) %></td>
                <td><%= session.agent_type %></td>
                <td><span class={"badge badge-#{session.status}"}><%= session.status %></span></td>
                <td><%= Path.basename(session.role_prompt_path || "n/a") %></td>
                <td class="mono text-muted"><%= short_path(session.workspace_path) %></td>
                <td><%= format_time(session.started_at) %></td>
                <td><%= format_time(session.last_active_at) %></td>
              </tr>
            <% end %>

            <%= if @sessions == [] do %>
              <tr><td colspan="7" class="text-muted text-center">No active sessions</td></tr>
            <% end %>
          </tbody>
        </table>
      </div>

      <div class="section">
        <h3>Recent Activity</h3>
        <div class="log-panel">
          <%= for log <- Enum.take(@logs, 50) do %>
            <div class="log-entry">
              <span class="log-time"><%= format_time(log.timestamp) %></span>
              <span class={"log-level log-#{log.level}"}><%= log.level %></span>
              <span class="log-message"><%= log.message %></span>
            </div>
          <% end %>

          <%= if @logs == [] do %>
            <div class="text-muted">No recent activity</div>
          <% end %>
        </div>
      </div>
    </div>
    """
  end

  @impl true
  def handle_info(:refresh, socket) do
    {:noreply, assign(socket, sessions: load_active_sessions())}
  end

  def handle_info(:observability_updated, socket) do
    {:noreply, assign(socket, sessions: load_active_sessions())}
  end

  def handle_info({:agent_output, session_id, msg}, socket) do
    log = %{
      timestamp: DateTime.utc_now(),
      level: "info",
      message: "#{session_id}: #{inspect(msg, limit: 100)}"
    }

    logs = [log | socket.assigns.logs] |> Enum.take(200)
    {:noreply, assign(socket, logs: logs)}
  end

  def handle_info(_msg, socket), do: {:noreply, socket}

  @impl true
  def handle_event("pause_agent", %{"session-id" => sid}, socket) do
    Anarchy.SessionManager.pause_session(sid, "manual")
    {:noreply, socket}
  end

  def handle_event("resume_agent", %{"session-id" => sid}, socket) do
    Anarchy.SessionManager.update_session(sid, %{status: "active", paused_at: nil})
    {:noreply, socket}
  end

  defp load_active_sessions do
    import Ecto.Query

    Anarchy.Schemas.AgentSession
    |> where([s], s.status in ["active", "paused", "resuming"])
    |> order_by(desc: :started_at)
    |> limit(100)
    |> Anarchy.Repo.all()
  rescue
    _ -> []
  end

  defp short_path(nil), do: "n/a"

  defp short_path(path) when is_binary(path) do
    parts = Path.split(path)

    if length(parts) > 3 do
      ".../" <> Enum.join(Enum.take(parts, -3), "/")
    else
      path
    end
  end

  defp format_time(nil), do: "-"
  defp format_time(%DateTime{} = dt), do: Calendar.strftime(dt, "%H:%M:%S")
  defp format_time(%NaiveDateTime{} = dt), do: Calendar.strftime(dt, "%H:%M:%S")
end
