defmodule Anarchy.Schemas.ProjectAssignment do
  @moduledoc false

  use Ecto.Schema

  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "project_assignments" do
    field :role, :string
    field :scope, :string
    field :agent_config, :map, default: %{}

    belongs_to :project, Anarchy.Schemas.Project

    has_many :tasks, Anarchy.Schemas.Task, foreign_key: :pm_assignment_id

    timestamps(updated_at: false)
  end

  @required_fields [:role, :project_id]
  @optional_fields [:scope, :agent_config]

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(project_assignment, attrs) do
    project_assignment
    |> cast(attrs, @required_fields ++ @optional_fields)
    |> validate_required(@required_fields)
    |> validate_length(:role, max: 50)
    |> validate_length(:scope, max: 500)
    |> foreign_key_constraint(:project_id)
  end
end
