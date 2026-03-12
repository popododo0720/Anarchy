defmodule Anarchy.DependencyGraph do
  @moduledoc "Task dependency visualization using Mermaid.js syntax."

  alias Anarchy.Projects

  @spec to_mermaid(Ecto.UUID.t()) :: String.t()
  def to_mermaid(project_id) do
    tasks = Projects.list_tasks(project_id)

    nodes =
      Enum.map_join(tasks, "\n", fn t ->
        style =
          case t.status do
            :completed -> ":::completed"
            s when s in [:working, :ce_reviewing, :code_reviewing, :planning] -> ":::active"
            :failed -> ":::failed"
            _ -> ""
          end

        "  #{short_id(t.id)}[\"#{escape_mermaid(t.title)}\"]#{style}"
      end)

    edges =
      tasks
      |> Enum.flat_map(fn t ->
        (t.depends_on || [])
        |> Enum.map(fn dep_id ->
          "  #{short_id(dep_id)} --> #{short_id(t.id)}"
        end)
      end)
      |> Enum.join("\n")

    """
    graph TD
    #{nodes}
    #{edges}
      classDef completed fill:#4ade80,stroke:#16a34a
      classDef active fill:#60a5fa,stroke:#2563eb
      classDef failed fill:#f87171,stroke:#dc2626
    """
  end

  defp short_id(uuid), do: String.slice(to_string(uuid), 0, 8)

  defp escape_mermaid(text) do
    text
    |> String.replace("\"", "&quot;")
    |> String.replace("<", "&lt;")
    |> String.replace(">", "&gt;")
  end
end
