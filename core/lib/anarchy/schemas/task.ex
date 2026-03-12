defmodule Anarchy.Schemas.Task do
  @moduledoc false

  use Ecto.Schema

  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "tasks" do
    field :title, :string
    field :description, :string
    field :role, :string
    field :status, Ecto.Enum, values: [:pending, :assigned, :planning, :plan_reviewing, :working, :ce_reviewing, :code_reviewing, :compounding, :completed, :failed], default: :pending
    field :priority, :integer, default: 5
    field :depends_on, {:array, Ecto.UUID}, default: []
    field :attempt, :integer, default: 0
    field :max_attempts, :integer, default: 3
    field :pr_url, :string
    field :branch, :string
    field :result, :map
    field :learnings, :string

    belongs_to :project, Anarchy.Schemas.Project
    belongs_to :design, Anarchy.Schemas.Design
    belongs_to :parent_task, Anarchy.Schemas.Task
    belongs_to :pm_assignment, Anarchy.Schemas.ProjectAssignment

    has_many :subtasks, Anarchy.Schemas.Task, foreign_key: :parent_task_id
    has_many :agent_sessions, Anarchy.Schemas.AgentSession

    timestamps()
  end

  @required_fields [:title, :role, :project_id]
  @optional_fields [
    :description,
    :status,
    :priority,
    :depends_on,
    :attempt,
    :max_attempts,
    :pr_url,
    :branch,
    :result,
    :learnings,
    :design_id,
    :parent_task_id,
    :pm_assignment_id
  ]

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(task, attrs) do
    task
    |> cast(attrs, @required_fields ++ @optional_fields)
    |> validate_required(@required_fields)
    |> validate_length(:title, max: 500)
    |> validate_length(:role, max: 50)
    |> validate_length(:pr_url, max: 500)
    |> validate_length(:branch, max: 255)
    |> validate_number(:priority, greater_than_or_equal_to: 0)
    |> validate_number(:max_attempts, greater_than: 0)
    |> foreign_key_constraint(:project_id)
    |> foreign_key_constraint(:design_id)
    |> foreign_key_constraint(:parent_task_id)
    |> foreign_key_constraint(:pm_assignment_id)
  end
end
