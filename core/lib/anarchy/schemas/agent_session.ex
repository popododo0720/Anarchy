defmodule Anarchy.Schemas.AgentSession do
  @moduledoc false

  use Ecto.Schema

  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "agent_sessions" do
    field :agent_type, :string
    field :session_id, :string
    field :role_prompt_path, :string
    field :workspace_path, :string
    field :branch, :string
    field :last_commit_sha, :string
    field :status, :string, default: "active"
    field :pause_reason, :string
    field :resume_context, :map
    field :started_at, :utc_datetime
    field :last_active_at, :utc_datetime
    field :paused_at, :utc_datetime
    field :ended_at, :utc_datetime

    belongs_to :task, Anarchy.Schemas.Task
    belongs_to :project, Anarchy.Schemas.Project
  end

  @required_fields [:agent_type, :session_id, :project_id, :started_at]
  @optional_fields [
    :task_id,
    :role_prompt_path,
    :workspace_path,
    :branch,
    :last_commit_sha,
    :status,
    :pause_reason,
    :resume_context,
    :last_active_at,
    :paused_at,
    :ended_at
  ]

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(agent_session, attrs) do
    agent_session
    |> cast(attrs, @required_fields ++ @optional_fields)
    |> validate_required(@required_fields)
    |> validate_length(:agent_type, max: 50)
    |> validate_length(:session_id, max: 255)
    |> validate_length(:role_prompt_path, max: 500)
    |> validate_length(:workspace_path, max: 500)
    |> validate_length(:branch, max: 255)
    |> validate_length(:last_commit_sha, max: 40)
    |> validate_length(:status, max: 50)
    |> validate_length(:pause_reason, max: 100)
    |> unique_constraint(:session_id)
    |> foreign_key_constraint(:task_id)
    |> foreign_key_constraint(:project_id)
  end
end
