defmodule AnarchyWeb.DesignEditorLive do
  use Phoenix.LiveView

  alias Anarchy.Projects

  @impl true
  def mount(%{"id" => id}, _session, socket) do
    design = Projects.get_design!(id)
    changeset = Projects.change_design(design)

    {:ok,
     assign(socket,
       page_title: "Design: #{design.title}",
       design: design,
       changeset: changeset,
       editing: false,
       preview: true
     )}
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="dashboard">
      <header class="dashboard-header">
        <h1>
          <a href="/projects" class="text-muted">Projects</a> /
          <a href={"/projects/#{@design.project_id}"} class="text-muted">Project</a> /
          <%= @design.title %>
        </h1>
        <div class="header-actions">
          <span class={"badge badge-#{@design.status}"}><%= @design.status %></span>
          <span class="text-muted">v<%= @design.version %></span>
        </div>
      </header>

      <div class="section">
        <div class="section-header">
          <div class="btn-group">
            <button phx-click="toggle_edit" class={"btn #{if @editing, do: "btn-primary"}"}>
              <%= if @editing, do: "Preview", else: "Edit" %>
            </button>
            <%= if @design.status == "draft" do %>
              <button phx-click="confirm" class="btn btn-primary">Confirm Design</button>
            <% end %>
          </div>
        </div>

        <%= if @editing do %>
          <.form for={@changeset} phx-submit="save">
            <div class="form-group">
              <label>Title</label>
              <input type="text" name="design[title]" value={@design.title} class="input" />
            </div>
            <div class="form-group">
              <label>Content (Markdown)</label>
              <textarea name="design[content_md]" rows="30" class="input mono"><%= @design.content_md %></textarea>
            </div>
            <button type="submit" class="btn btn-primary">Save Changes</button>
          </.form>
        <% else %>
          <div class="card">
            <div class="design-content">
              <pre class="mono"><%= @design.content_md %></pre>
            </div>
          </div>
        <% end %>
      </div>
    </div>
    """
  end

  @impl true
  def handle_event("toggle_edit", _params, socket) do
    {:noreply, assign(socket, editing: !socket.assigns.editing)}
  end

  def handle_event("save", %{"design" => params}, socket) do
    case Projects.update_design(socket.assigns.design, params) do
      {:ok, design} ->
        {:noreply,
         socket
         |> assign(design: design, editing: false)
         |> put_flash(:info, "Design saved")}

      {:error, changeset} ->
        {:noreply, assign(socket, changeset: changeset)}
    end
  end

  def handle_event("confirm", _params, socket) do
    case Projects.confirm_design(socket.assigns.design) do
      {:ok, design} ->
        {:noreply,
         socket
         |> assign(design: design)
         |> put_flash(:info, "Design confirmed")}

      {:error, _changeset} ->
        {:noreply, put_flash(socket, :error, "Failed to confirm design")}
    end
  end
end
