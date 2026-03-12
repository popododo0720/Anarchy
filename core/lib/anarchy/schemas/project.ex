defmodule Anarchy.Schemas.Project do
  @moduledoc false

  use Ecto.Schema

  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "projects" do
    field :name, :string
    field :description, :string
    field :status, :string, default: "active"
    field :repo_url, :string
    field :base_branch, :string, default: "main"
    field :config, :map, default: %{}

    has_many :designs, Anarchy.Schemas.Design
    has_many :project_assignments, Anarchy.Schemas.ProjectAssignment
    has_many :tasks, Anarchy.Schemas.Task
    has_many :agent_sessions, Anarchy.Schemas.AgentSession

    timestamps()
  end

  @required_fields [:name]
  @optional_fields [:description, :status, :repo_url, :base_branch, :config]

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(project, attrs) do
    project
    |> cast(attrs, @required_fields ++ @optional_fields)
    |> validate_required(@required_fields)
    |> validate_length(:name, max: 255)
    |> validate_length(:repo_url, max: 500)
    |> validate_length(:base_branch, max: 255)
    |> validate_length(:status, max: 50)
  end
end
