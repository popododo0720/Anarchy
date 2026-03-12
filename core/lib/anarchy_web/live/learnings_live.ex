defmodule AnarchyWeb.LearningsLive do
  use Phoenix.LiveView

  @impl true
  def mount(_params, _session, socket) do
    learnings = load_all_learnings()
    {:ok, assign(socket, page_title: "Learnings", learnings: learnings, search: "", filtered: learnings)}
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="dashboard">
      <header class="dashboard-header">
        <h1>Compound Learnings</h1>
      </header>

      <div class="section">
        <form phx-change="search" class="search-form">
          <input type="text" name="query" value={@search} placeholder="Search learnings..." class="input" phx-debounce="300" />
        </form>
      </div>

      <div class="learnings-list">
        <%= for l <- @filtered do %>
          <div class="card learning-card">
            <div class="card-header">
              <strong><%= l.filename %></strong>
              <span class="text-muted"><%= format_time(l.modified_at) %></span>
            </div>
            <div class="card-body">
              <pre class="learning-content"><%= l.content %></pre>
            </div>
          </div>
        <% end %>

        <%= if @filtered == [] do %>
          <div class="empty-state">
            <%= if @search != "" do %>
              <p>No learnings match "<%= @search %>"</p>
            <% else %>
              <p>No learnings documented yet. Complete CE loops to generate learnings.</p>
            <% end %>
          </div>
        <% end %>
      </div>
    </div>
    """
  end

  @impl true
  def handle_event("search", %{"query" => query}, socket) do
    filtered =
      if query == "" do
        socket.assigns.learnings
      else
        Enum.filter(socket.assigns.learnings, fn l ->
          String.contains?(String.downcase(l.content), String.downcase(query)) ||
            String.contains?(String.downcase(l.filename), String.downcase(query))
        end)
      end

    {:noreply, assign(socket, search: query, filtered: filtered)}
  end

  defp load_all_learnings do
    # Check both workspace docs/solutions and DB task learnings
    file_learnings = load_file_learnings()
    db_learnings = load_db_learnings()
    (file_learnings ++ db_learnings) |> Enum.sort_by(& &1.modified_at, {:desc, NaiveDateTime})
  end

  defp load_file_learnings do
    Path.wildcard("docs/solutions/**/*.md")
    |> Enum.flat_map(fn path ->
      case File.stat(path) do
        {:ok, stat} ->
          [%{
            path: path,
            filename: Path.basename(path),
            content: File.read!(path),
            modified_at: stat.mtime |> NaiveDateTime.from_erl!()
          }]
        {:error, _} -> []
      end
    end)
  end

  defp load_db_learnings do
    import Ecto.Query
    Anarchy.Schemas.Task
    |> where([t], not is_nil(t.learnings))
    |> select([t], %{id: t.id, title: t.title, learnings: t.learnings, updated_at: t.updated_at})
    |> Anarchy.Repo.all()
    |> Enum.map(fn t ->
      %{
        path: nil,
        filename: "Task: #{t.title}",
        content: t.learnings,
        modified_at: t.updated_at
      }
    end)
  end

  defp format_time({{y, m, d}, {h, min, _s}}), do: "#{y}-#{pad(m)}-#{pad(d)} #{pad(h)}:#{pad(min)}"
  defp format_time(%NaiveDateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d %H:%M")
  defp format_time(_), do: ""

  defp pad(n), do: String.pad_leading(to_string(n), 2, "0")
end
