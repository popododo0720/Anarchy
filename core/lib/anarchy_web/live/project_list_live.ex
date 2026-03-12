defmodule AnarchyWeb.ProjectListLive do
  use Phoenix.LiveView

  alias Anarchy.Projects

  @impl true
  def mount(_params, _session, socket) do
    projects = Projects.list_projects()

    {:ok,
     assign(socket,
       page_title: "Projects",
       projects: projects,
       show_form: false,
       changeset: Projects.change_project(%Anarchy.Schemas.Project{})
     )}
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="dashboard">
      <header class="dashboard-header">
        <h1>Anarchy Projects</h1>
        <nav class="nav-links">
          <a href="/">Monitor</a>
          <a href="/projects" class="active">Projects</a>
        </nav>
      </header>

      <div class="section">
        <div class="section-header">
          <h2>All Projects</h2>
          <button phx-click="toggle_form" class="btn btn-primary">
            <%= if @show_form, do: "Cancel", else: "New Project" %>
          </button>
        </div>

        <%= if @show_form do %>
          <div class="card form-card">
            <.form for={@changeset} phx-submit="create_project">
              <div class="form-group">
                <label>Name</label>
                <input type="text" name="project[name]" value={@changeset.changes[:name]} required class="input" />
              </div>
              <div class="form-group">
                <label>Description</label>
                <textarea name="project[description]" rows="3" class="input"><%= @changeset.changes[:description] %></textarea>
              </div>
              <div class="form-row">
                <div class="form-group">
                  <label>Repository URL</label>
                  <input type="text" name="project[repo_url]" value={@changeset.changes[:repo_url]} class="input" placeholder="https://github.com/..." />
                </div>
                <div class="form-group">
                  <label>Base Branch</label>
                  <input type="text" name="project[base_branch]" value={@changeset.changes[:base_branch] || "main"} class="input" />
                </div>
              </div>
              <button type="submit" class="btn btn-primary">Create Project</button>
            </.form>
          </div>
        <% end %>

        <div class="project-grid">
          <%= for project <- @projects do %>
            <a href={"/projects/#{project.id}"} class="card project-card">
              <div class="project-name"><%= project.name %></div>
              <div class="project-desc"><%= project.description || "No description" %></div>
              <div class="project-meta">
                <span class={"badge badge-#{project.status}"}><%= project.status %></span>
                <span class="text-muted"><%= format_date(project.updated_at) %></span>
              </div>
            </a>
          <% end %>

          <%= if @projects == [] do %>
            <div class="empty-state">
              <p>No projects yet. Create one to get started.</p>
            </div>
          <% end %>
        </div>
      </div>
    </div>
    """
  end

  @impl true
  def handle_event("toggle_form", _params, socket) do
    {:noreply, assign(socket, show_form: !socket.assigns.show_form)}
  end

  def handle_event("create_project", %{"project" => params}, socket) do
    case Projects.create_project(params) do
      {:ok, _project} ->
        {:noreply,
         socket
         |> assign(projects: Projects.list_projects(), show_form: false)
         |> put_flash(:info, "Project created")}

      {:error, changeset} ->
        {:noreply, assign(socket, changeset: changeset)}
    end
  end

  defp format_date(nil), do: ""

  defp format_date(%NaiveDateTime{} = dt) do
    Calendar.strftime(dt, "%Y-%m-%d %H:%M")
  end

  defp format_date(%DateTime{} = dt) do
    Calendar.strftime(dt, "%Y-%m-%d %H:%M")
  end
end
