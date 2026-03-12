defmodule AnarchyWeb.ProjectDetailLive do
  use Phoenix.LiveView

  alias Anarchy.{Projects, PMAgent}

  @impl true
  def mount(%{"id" => id}, _session, socket) do
    project = Projects.get_project!(id)
    designs = Projects.list_designs(id)
    tasks = Projects.list_tasks(id)
    stats = Projects.project_stats(id)

    if connected?(socket) do
      Phoenix.PubSub.subscribe(Anarchy.PubSub, "project:#{id}")
    end

    {:ok,
     assign(socket,
       page_title: project.name,
       project: project,
       designs: designs,
       tasks: tasks,
       stats: stats,
       tab: "overview",
       show_design_form: false,
       design_changeset: Projects.change_design(%Anarchy.Schemas.Design{})
     )}
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="dashboard">
      <header class="dashboard-header">
        <h1><a href="/projects" class="text-muted">Projects</a> / <%= @project.name %></h1>
        <span class={"badge badge-#{@project.status}"}><%= @project.status %></span>
      </header>

      <div class="tab-bar">
        <button phx-click="set_tab" phx-value-tab="overview" class={"tab #{if @tab == "overview", do: "active"}"}> Overview </button>
        <button phx-click="set_tab" phx-value-tab="designs" class={"tab #{if @tab == "designs", do: "active"}"}> Designs (<%= length(@designs) %>) </button>
        <button phx-click="set_tab" phx-value-tab="tasks" class={"tab #{if @tab == "tasks", do: "active"}"}> Tasks (<%= length(@tasks) %>) </button>
      </div>

      <%= case @tab do %>
        <% "overview" -> %>
          <div class="metrics-row">
            <div class="metric-card"><div class="metric-value"><%= @stats.total_tasks %></div><div class="metric-label">Total Tasks</div></div>
            <div class="metric-card"><div class="metric-value"><%= @stats.completed_tasks %></div><div class="metric-label">Completed</div></div>
            <div class="metric-card"><div class="metric-value"><%= @stats.active_tasks %></div><div class="metric-label">Active</div></div>
            <div class="metric-card"><div class="metric-value"><%= @stats.active_sessions %></div><div class="metric-label">Sessions</div></div>
          </div>

          <div class="section">
            <h3>Recent Tasks</h3>
            <table class="data-table">
              <thead><tr><th>Title</th><th>Role</th><th>Status</th><th>Priority</th></tr></thead>
              <tbody>
                <%= for task <- Enum.take(@tasks, 10) do %>
                  <tr>
                    <td><a href={"/tasks/#{task.id}"}><%= task.title %></a></td>
                    <td><%= task.role %></td>
                    <td><span class={"badge badge-#{task.status}"}><%= task.status %></span></td>
                    <td><%= task.priority %></td>
                  </tr>
                <% end %>
              </tbody>
            </table>
          </div>

        <% "designs" -> %>
          <div class="section">
            <div class="section-header">
              <h3>Designs</h3>
              <button phx-click="toggle_design_form" class="btn btn-primary">
                <%= if @show_design_form, do: "Cancel", else: "New Design" %>
              </button>
            </div>

            <%= if @show_design_form do %>
              <div class="card form-card">
                <.form for={@design_changeset} phx-submit="create_design">
                  <input type="hidden" name="design[project_id]" value={@project.id} />
                  <div class="form-group">
                    <label>Title</label>
                    <input type="text" name="design[title]" required class="input" />
                  </div>
                  <div class="form-group">
                    <label>Content (Markdown)</label>
                    <textarea name="design[content_md]" rows="15" required class="input mono"><%= @design_changeset.changes[:content_md] %></textarea>
                  </div>
                  <button type="submit" class="btn btn-primary">Create Design</button>
                </.form>
              </div>
            <% end %>

            <%= for design <- @designs do %>
              <div class="card">
                <div class="card-header">
                  <a href={"/designs/#{design.id}"}><strong><%= design.title %></strong></a>
                  <span class={"badge badge-#{design.status}"}><%= design.status %></span>
                </div>
                <div class="card-body text-muted">
                  v<%= design.version %> · <%= format_date(design.updated_at) %>
                </div>
                <%= if design.status == "draft" do %>
                  <div class="card-actions">
                    <button phx-click="confirm_design" phx-value-id={design.id} class="btn btn-sm">Confirm</button>
                  </div>
                <% end %>
                <%= if design.status == "confirmed" do %>
                  <div class="card-actions">
                    <button phx-click="decompose_design" phx-value-id={design.id} class="btn btn-sm btn-primary">Decompose → Tasks</button>
                  </div>
                <% end %>
              </div>
            <% end %>
          </div>

        <% "tasks" -> %>
          <div class="section">
            <h3>All Tasks</h3>
            <table class="data-table">
              <thead><tr><th>Title</th><th>Role</th><th>Status</th><th>Priority</th><th>Attempt</th></tr></thead>
              <tbody>
                <%= for task <- @tasks do %>
                  <tr>
                    <td><a href={"/tasks/#{task.id}"}><%= task.title %></a></td>
                    <td><%= task.role %></td>
                    <td><span class={"badge badge-#{task.status}"}><%= task.status %></span></td>
                    <td><%= task.priority %></td>
                    <td><%= task.attempt %>/<%= task.max_attempts %></td>
                  </tr>
                <% end %>
              </tbody>
            </table>
          </div>
      <% end %>
    </div>
    """
  end

  @impl true
  def handle_event("set_tab", %{"tab" => tab}, socket) do
    {:noreply, assign(socket, tab: tab)}
  end

  def handle_event("toggle_design_form", _params, socket) do
    {:noreply, assign(socket, show_design_form: !socket.assigns.show_design_form)}
  end

  def handle_event("create_design", %{"design" => params}, socket) do
    case Projects.create_design(params) do
      {:ok, _design} ->
        designs = Projects.list_designs(socket.assigns.project.id)
        {:noreply, assign(socket, designs: designs, show_design_form: false)}

      {:error, changeset} ->
        {:noreply, assign(socket, design_changeset: changeset)}
    end
  end

  def handle_event("confirm_design", %{"id" => id}, socket) do
    design = Projects.get_design!(id)

    if design.project_id != socket.assigns.project.id do
      {:noreply, put_flash(socket, :error, "Design not found")}
    else
    case Projects.confirm_design(design) do
      {:ok, confirmed_design} ->
        # Async decompose — results arrive via PubSub
        case PMAgent.decompose_async(confirmed_design) do
          {:ok, _pid} ->
            designs = Projects.list_designs(socket.assigns.project.id)

            {:noreply,
             socket
             |> assign(designs: designs)
             |> put_flash(:info, "Design confirmed — decomposing into tasks...")}

          {:error, _reason} ->
            designs = Projects.list_designs(socket.assigns.project.id)
            {:noreply, assign(socket, designs: designs) |> put_flash(:info, "Design confirmed (auto-decompose skipped)")}
        end

      {:error, _changeset} ->
        {:noreply, put_flash(socket, :error, "Failed to confirm design")}
    end
    end
  end

  def handle_event("decompose_design", %{"id" => id}, socket) do
    design = Projects.get_design!(id)

    if design.project_id != socket.assigns.project.id do
      {:noreply, put_flash(socket, :error, "Design not found")}
    else
    case PMAgent.decompose_async(design) do
      {:ok, _pid} ->
        {:noreply, put_flash(socket, :info, "Decomposing design into tasks...")}

      {:error, _reason} ->
        {:noreply, put_flash(socket, :error, "Decomposition failed. Check server logs.")}
    end
    end
  end

  @impl true
  def handle_info({:tasks_created, _design_id, _tasks}, socket) do
    tasks = Projects.list_tasks(socket.assigns.project.id)
    stats = Projects.project_stats(socket.assigns.project.id)

    {:noreply,
     socket
     |> assign(tasks: tasks, stats: stats, tab: "tasks")
     |> put_flash(:info, "Tasks created from design")}
  end

  def handle_info({:pm_error, _design_id, _reason}, socket) do
    {:noreply, put_flash(socket, :error, "PM decomposition failed")}
  end

  def handle_info({:task_status_changed, _task_id, _new_status}, socket) do
    tasks = Projects.list_tasks(socket.assigns.project.id)
    stats = Projects.project_stats(socket.assigns.project.id)
    {:noreply, assign(socket, tasks: tasks, stats: stats)}
  end

  def handle_info(_msg, socket), do: {:noreply, socket}

  defp format_date(nil), do: ""
  defp format_date(%NaiveDateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d %H:%M")
  defp format_date(%DateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d %H:%M")
end
