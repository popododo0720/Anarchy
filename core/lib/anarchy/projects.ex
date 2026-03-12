defmodule Anarchy.Projects do
  @moduledoc """
  Context module for project, design, and task CRUD operations.
  """

  import Ecto.Query

  alias Anarchy.Repo
  alias Anarchy.Schemas.{Project, Design, Task, AgentSession, ProjectAssignment}

  # --- Projects ---

  def list_projects do
    Project
    |> order_by(desc: :updated_at)
    |> Repo.all()
  end

  def get_project!(id), do: Repo.get!(Project, id)

  def get_project(id), do: Repo.get(Project, id)

  def create_project(attrs) do
    %Project{}
    |> Project.changeset(attrs)
    |> Repo.insert()
  end

  def update_project(%Project{} = project, attrs) do
    project
    |> Project.changeset(attrs)
    |> Repo.update()
  end

  def delete_project(%Project{} = project) do
    Repo.delete(project)
  end

  def change_project(%Project{} = project, attrs \\ %{}) do
    Project.changeset(project, attrs)
  end

  # --- Designs ---

  def list_designs(project_id) do
    Design
    |> where([d], d.project_id == ^project_id)
    |> order_by(desc: :updated_at)
    |> Repo.all()
  end

  def get_design!(id), do: Repo.get!(Design, id)

  def create_design(attrs) do
    %Design{}
    |> Design.changeset(attrs)
    |> Repo.insert()
  end

  def update_design(%Design{} = design, attrs) do
    design
    |> Design.changeset(attrs)
    |> Repo.update()
  end

  def confirm_design(%Design{} = design) do
    design
    |> Design.changeset(%{status: "confirmed", confirmed_at: DateTime.utc_now()})
    |> Repo.update()
  end

  def change_design(%Design{} = design, attrs \\ %{}) do
    Design.changeset(design, attrs)
  end

  # --- Tasks ---

  def list_tasks(project_id) do
    Task
    |> where([t], t.project_id == ^project_id)
    |> order_by([t], [asc: t.priority, asc: t.inserted_at])
    |> Repo.all()
  end

  def list_tasks_by_status(project_id, status) when is_atom(status) do
    Task
    |> where([t], t.project_id == ^project_id and t.status == ^status)
    |> order_by([t], [asc: t.priority, asc: t.inserted_at])
    |> Repo.all()
  end

  def get_task!(id), do: Repo.get!(Task, id)

  def get_task(id), do: Repo.get(Task, id)

  def create_task(attrs) do
    %Task{}
    |> Task.changeset(attrs)
    |> Repo.insert()
  end

  def update_task(%Task{} = task, attrs) do
    task
    |> Task.changeset(attrs)
    |> Repo.update()
  end

  def change_task(%Task{} = task, attrs \\ %{}) do
    Task.changeset(task, attrs)
  end

  def task_counts_by_status(project_id) do
    Task
    |> where([t], t.project_id == ^project_id)
    |> group_by([t], t.status)
    |> select([t], {t.status, count(t.id)})
    |> Repo.all()
    |> Map.new()
  end

  # --- Agent Sessions ---

  def list_sessions(project_id) do
    AgentSession
    |> where([s], s.project_id == ^project_id)
    |> order_by(desc: :started_at)
    |> Repo.all()
  end

  def active_sessions(project_id) do
    AgentSession
    |> where([s], s.project_id == ^project_id and s.status == "active")
    |> order_by(desc: :started_at)
    |> Repo.all()
  end

  # --- Project Assignments ---

  def list_assignments(project_id) do
    ProjectAssignment
    |> where([a], a.project_id == ^project_id)
    |> Repo.all()
  end

  def create_assignment(attrs) do
    %ProjectAssignment{}
    |> ProjectAssignment.changeset(attrs)
    |> Repo.insert()
  end

  # --- Stats ---

  def project_stats(project_id) do
    tasks = list_tasks(project_id)
    sessions = list_sessions(project_id)

    %{
      total_tasks: length(tasks),
      completed_tasks: Enum.count(tasks, &(&1.status == :completed)),
      failed_tasks: Enum.count(tasks, &(&1.status == :failed)),
      active_tasks: Enum.count(tasks, &(&1.status in [:working, :planning, :ce_reviewing, :code_reviewing])),
      active_sessions: Enum.count(sessions, &(&1.status == "active")),
      total_sessions: length(sessions)
    }
  end
end
