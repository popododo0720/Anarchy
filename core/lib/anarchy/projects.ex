defmodule Anarchy.Projects do
  @moduledoc """
  Context module for project, design, and task CRUD operations.
  """

  import Ecto.Query

  alias Anarchy.Repo
  alias Anarchy.Schemas.{Project, Design, Task, AgentSession, ProjectAssignment}

  # --- Projects ---

  @spec list_projects() :: [Project.t()]
  def list_projects do
    Project
    |> order_by(desc: :updated_at)
    |> Repo.all()
  end

  @spec get_project!(Ecto.UUID.t()) :: Project.t()
  def get_project!(id), do: Repo.get!(Project, id)

  @spec get_project(Ecto.UUID.t()) :: Project.t() | nil
  def get_project(id), do: Repo.get(Project, id)

  @spec create_project(map()) :: {:ok, Project.t()} | {:error, Ecto.Changeset.t()}
  def create_project(attrs) do
    %Project{}
    |> Project.changeset(attrs)
    |> Repo.insert()
  end

  @spec update_project(Project.t(), map()) :: {:ok, Project.t()} | {:error, Ecto.Changeset.t()}
  def update_project(%Project{} = project, attrs) do
    project
    |> Project.changeset(attrs)
    |> Repo.update()
  end

  @spec delete_project(Project.t()) :: {:ok, Project.t()} | {:error, Ecto.Changeset.t()}
  def delete_project(%Project{} = project) do
    Repo.delete(project)
  end

  @spec change_project(Project.t(), map()) :: Ecto.Changeset.t()
  def change_project(%Project{} = project, attrs \\ %{}) do
    Project.changeset(project, attrs)
  end

  # --- Designs ---

  @spec list_designs(Ecto.UUID.t()) :: [Design.t()]
  def list_designs(project_id) do
    Design
    |> where([d], d.project_id == ^project_id)
    |> order_by(desc: :updated_at)
    |> Repo.all()
  end

  @spec get_design!(Ecto.UUID.t()) :: Design.t()
  def get_design!(id), do: Repo.get!(Design, id)

  @spec create_design(map()) :: {:ok, Design.t()} | {:error, Ecto.Changeset.t()}
  def create_design(attrs) do
    %Design{}
    |> Design.changeset(attrs)
    |> Repo.insert()
  end

  @spec update_design(Design.t(), map()) :: {:ok, Design.t()} | {:error, Ecto.Changeset.t()}
  def update_design(%Design{} = design, attrs) do
    design
    |> Design.changeset(attrs)
    |> Repo.update()
  end

  @spec confirm_design(Design.t()) :: {:ok, Design.t()} | {:error, Ecto.Changeset.t()}
  def confirm_design(%Design{} = design) do
    design
    |> Design.changeset(%{status: "confirmed", confirmed_at: DateTime.utc_now()})
    |> Repo.update()
  end

  @spec change_design(Design.t(), map()) :: Ecto.Changeset.t()
  def change_design(%Design{} = design, attrs \\ %{}) do
    Design.changeset(design, attrs)
  end

  # --- Tasks ---

  @spec list_tasks(Ecto.UUID.t()) :: [Task.t()]
  def list_tasks(project_id) do
    Task
    |> where([t], t.project_id == ^project_id)
    |> order_by([t], [asc: t.priority, asc: t.inserted_at])
    |> Repo.all()
  end

  @spec list_tasks_by_status(Ecto.UUID.t(), atom()) :: [Task.t()]
  def list_tasks_by_status(project_id, status) when is_atom(status) do
    Task
    |> where([t], t.project_id == ^project_id and t.status == ^status)
    |> order_by([t], [asc: t.priority, asc: t.inserted_at])
    |> Repo.all()
  end

  @spec get_task!(Ecto.UUID.t()) :: Task.t()
  def get_task!(id), do: Repo.get!(Task, id)

  @spec get_task(Ecto.UUID.t()) :: Task.t() | nil
  def get_task(id), do: Repo.get(Task, id)

  @spec create_task(map()) :: {:ok, Task.t()} | {:error, Ecto.Changeset.t()}
  def create_task(attrs) do
    %Task{}
    |> Task.changeset(attrs)
    |> Repo.insert()
  end

  @spec update_task(Task.t(), map()) :: {:ok, Task.t()} | {:error, Ecto.Changeset.t()}
  def update_task(%Task{} = task, attrs) do
    task
    |> Task.changeset(attrs)
    |> Repo.update()
  end

  @spec change_task(Task.t(), map()) :: Ecto.Changeset.t()
  def change_task(%Task{} = task, attrs \\ %{}) do
    Task.changeset(task, attrs)
  end

  @spec task_counts_by_status(Ecto.UUID.t()) :: %{atom() => non_neg_integer()}
  def task_counts_by_status(project_id) do
    Task
    |> where([t], t.project_id == ^project_id)
    |> group_by([t], t.status)
    |> select([t], {t.status, count(t.id)})
    |> Repo.all()
    |> Map.new()
  end

  # --- Agent Sessions ---

  @spec list_sessions(Ecto.UUID.t()) :: [AgentSession.t()]
  def list_sessions(project_id) do
    AgentSession
    |> where([s], s.project_id == ^project_id)
    |> order_by(desc: :started_at)
    |> Repo.all()
  end

  @spec active_sessions(Ecto.UUID.t()) :: [AgentSession.t()]
  def active_sessions(project_id) do
    AgentSession
    |> where([s], s.project_id == ^project_id and s.status == "active")
    |> order_by(desc: :started_at)
    |> Repo.all()
  end

  # --- Project Assignments ---

  @spec list_assignments(Ecto.UUID.t()) :: [ProjectAssignment.t()]
  def list_assignments(project_id) do
    ProjectAssignment
    |> where([a], a.project_id == ^project_id)
    |> Repo.all()
  end

  @spec create_assignment(map()) :: {:ok, ProjectAssignment.t()} | {:error, Ecto.Changeset.t()}
  def create_assignment(attrs) do
    %ProjectAssignment{}
    |> ProjectAssignment.changeset(attrs)
    |> Repo.insert()
  end

  # --- Stats ---

  @spec project_stats(Ecto.UUID.t()) :: map()
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
