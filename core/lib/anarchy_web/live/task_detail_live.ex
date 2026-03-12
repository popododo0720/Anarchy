defmodule AnarchyWeb.TaskDetailLive do
  use Phoenix.LiveView

  alias Anarchy.{Projects, WorkflowEngine}

  @impl true
  def mount(%{"id" => id}, _session, socket) do
    task = Projects.get_task!(id)

    if connected?(socket) do
      Phoenix.PubSub.subscribe(Anarchy.PubSub, "task:#{id}")
    end

    {:ok,
     assign(socket,
       page_title: "Task: #{task.title}",
       task: task,
       ce_state: nil,
       sessions: list_task_sessions(task)
     )}
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="dashboard">
      <header class="dashboard-header">
        <h1>
          <a href="/projects" class="text-muted">Projects</a> /
          <a href={"/projects/#{@task.project_id}"} class="text-muted">Project</a> /
          <%= @task.title %>
        </h1>
        <span class={"badge badge-#{@task.status}"}><%= @task.status %></span>
      </header>

      <div class="metrics-row">
        <div class="metric-card"><div class="metric-value"><%= @task.role %></div><div class="metric-label">Role</div></div>
        <div class="metric-card"><div class="metric-value"><%= @task.priority %></div><div class="metric-label">Priority</div></div>
        <div class="metric-card"><div class="metric-value"><%= @task.attempt %>/<%= @task.max_attempts %></div><div class="metric-label">Attempts</div></div>
        <div class="metric-card"><div class="metric-value"><%= length(@sessions) %></div><div class="metric-label">Sessions</div></div>
      </div>

      <div class="section">
        <h3>Description</h3>
        <div class="card">
          <pre class="mono"><%= @task.description || "No description" %></pre>
        </div>
      </div>

      <div class="section">
        <h3>CE Loop</h3>
        <div class="ce-loop-progress">
          <%= for step <- ce_steps() do %>
            <div class={"ce-step #{step_class(step, @task.status)}"}>
              <div class="ce-step-name"><%= step %></div>
            </div>
          <% end %>
        </div>

        <div class="card-actions">
          <%= if @task.status == :pending do %>
            <button phx-click="start_ce_loop" class="btn btn-primary">Start CE Loop</button>
          <% end %>
          <%= if @task.status in [:working, :planning, :ce_reviewing, :code_reviewing] do %>
            <button phx-click="cancel_ce_loop" class="btn btn-danger">Cancel</button>
          <% end %>
        </div>
      </div>

      <%= if @task.depends_on != [] do %>
        <div class="section">
          <h3>Dependencies</h3>
          <ul>
            <%= for dep_id <- @task.depends_on do %>
              <li><a href={"/tasks/#{dep_id}"}><%= dep_id %></a></li>
            <% end %>
          </ul>
        </div>
      <% end %>

      <%= if @task.learnings do %>
        <div class="section">
          <h3>Learnings</h3>
          <div class="card">
            <pre class="mono"><%= @task.learnings %></pre>
          </div>
        </div>
      <% end %>

      <%= if @task.pr_url do %>
        <div class="section">
          <h3>Pull Request</h3>
          <a href={@task.pr_url} class="btn"><%= @task.pr_url %></a>
        </div>
      <% end %>

      <div class="section">
        <h3>Agent Sessions</h3>
        <table class="data-table">
          <thead><tr><th>Session ID</th><th>Type</th><th>Status</th><th>Started</th></tr></thead>
          <tbody>
            <%= for session <- @sessions do %>
              <tr>
                <td class="mono"><%= String.slice(session.session_id, 0, 20) %></td>
                <td><%= session.agent_type %></td>
                <td><span class={"badge badge-#{session.status}"}><%= session.status %></span></td>
                <td><%= format_date(session.started_at) %></td>
              </tr>
            <% end %>
          </tbody>
        </table>
      </div>
    </div>
    """
  end

  @impl true
  def handle_event("start_ce_loop", _params, socket) do
    task = socket.assigns.task

    case Anarchy.Workers.CELoopWorker.enqueue(task.id, task.project_id) do
      {:ok, _job} ->
        {:noreply, put_flash(socket, :info, "CE loop started")}

      {:error, reason} ->
        {:noreply, put_flash(socket, :error, "Failed to start CE loop: #{inspect(reason)}")}
    end
  end

  def handle_event("cancel_ce_loop", _params, socket) do
    task = socket.assigns.task

    try do
      WorkflowEngine.current_state({:global, {WorkflowEngine, task.id}})
      WorkflowEngine.trigger({:global, {WorkflowEngine, task.id}}, :cancel)
      {:noreply, put_flash(socket, :info, "CE loop cancelled")}
    rescue
      _ -> {:noreply, put_flash(socket, :error, "No active CE loop found")}
    end
  end

  @impl true
  def handle_info({:task_updated, task}, socket) do
    {:noreply, assign(socket, task: task)}
  end

  def handle_info(_msg, socket), do: {:noreply, socket}

  defp ce_steps do
    ~w(planning plan_reviewing working ce_reviewing code_reviewing compounding completed)
  end

  defp step_class(step, current_status) do
    current = Atom.to_string(current_status)
    steps = ce_steps()
    current_idx = Enum.find_index(steps, &(&1 == current)) || -1
    step_idx = Enum.find_index(steps, &(&1 == step)) || 99

    cond do
      step == current -> "active"
      step_idx < current_idx -> "done"
      true -> "pending"
    end
  end

  defp list_task_sessions(task) do
    import Ecto.Query
    Anarchy.Schemas.AgentSession
    |> where([s], s.task_id == ^task.id)
    |> order_by(desc: :started_at)
    |> Anarchy.Repo.all()
  rescue
    _ -> []
  end

  defp format_date(nil), do: ""
  defp format_date(%DateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d %H:%M")
  defp format_date(%NaiveDateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d %H:%M")
end
