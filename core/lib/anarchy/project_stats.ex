defmodule Anarchy.ProjectStats do
  @moduledoc "Enhanced project statistics."

  alias Anarchy.{Projects, SessionManager}

  @spec compute(Ecto.UUID.t()) :: map()
  def compute(project_id) do
    tasks = Projects.list_tasks(project_id)
    sessions = SessionManager.active_sessions_for_project(project_id)

    %{
      total_tasks: length(tasks),
      completed: Enum.count(tasks, &(&1.status == :completed)),
      in_progress: Enum.count(tasks, &(&1.status in [:working, :ce_reviewing, :code_reviewing, :planning, :plan_reviewing, :compounding])),
      failed: Enum.count(tasks, &(&1.status == :failed)),
      pending: Enum.count(tasks, &(&1.status == :pending)),
      completion_rate: completion_rate(tasks),
      active_agents: length(sessions),
      total_ce_loops: total_ce_loops(tasks),
      avg_attempts: avg_attempts(tasks)
    }
  end

  defp completion_rate(tasks) do
    total = length(tasks)
    if total == 0, do: 0.0, else: Float.round(Enum.count(tasks, &(&1.status == :completed)) / total * 100, 1)
  end

  defp total_ce_loops(tasks) do
    Enum.sum(Enum.map(tasks, & &1.attempt))
  end

  defp avg_attempts(tasks) do
    completed = Enum.filter(tasks, &(&1.status == :completed))
    if completed == [], do: 0.0, else: Float.round(Enum.sum(Enum.map(completed, & &1.attempt)) / length(completed), 1)
  end
end
