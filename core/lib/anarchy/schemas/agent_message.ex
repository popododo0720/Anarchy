defmodule Anarchy.Schemas.AgentMessage do
  @moduledoc false

  use Ecto.Schema
  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "agent_messages" do
    field :from_agent, :string
    field :to_agent, :string
    field :thread_id, Ecto.UUID
    field :subject, :string
    field :body, :string
    field :type, :string
    field :priority, :string, default: "normal"
    field :payload, :map
    field :read_at, :utc_datetime
    field :inserted_at, :utc_datetime

    belongs_to :project, Anarchy.Schemas.Project
  end

  @required [:from_agent, :subject, :body, :type, :project_id]
  @optional [:to_agent, :thread_id, :priority, :payload, :read_at, :inserted_at]

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(message, attrs) do
    message
    |> cast(attrs, @required ++ @optional)
    |> validate_required(@required)
    |> validate_inclusion(:type, ~w(status question result error assign review_request worker_done merge_ready escalation health_check))
    |> validate_inclusion(:priority, ~w(low normal high urgent))
    |> validate_length(:from_agent, max: 255)
    |> validate_length(:to_agent, max: 255)
    |> validate_length(:subject, max: 500)
    |> foreign_key_constraint(:project_id)
    |> put_inserted_at()
  end

  defp put_inserted_at(changeset) do
    if get_field(changeset, :inserted_at) do
      changeset
    else
      put_change(changeset, :inserted_at, DateTime.utc_now())
    end
  end
end
