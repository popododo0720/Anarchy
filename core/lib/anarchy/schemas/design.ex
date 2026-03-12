defmodule Anarchy.Schemas.Design do
  @moduledoc false

  use Ecto.Schema

  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "designs" do
    field :title, :string
    field :content_md, :string
    field :status, :string, default: "draft"
    field :version, :integer, default: 1
    field :confirmed_at, :utc_datetime

    belongs_to :project, Anarchy.Schemas.Project

    timestamps()
  end

  @required_fields [:title, :content_md, :project_id]
  @optional_fields [:status, :version, :confirmed_at]

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(design, attrs) do
    design
    |> cast(attrs, @required_fields ++ @optional_fields)
    |> validate_required(@required_fields)
    |> validate_length(:title, max: 500)
    |> validate_length(:status, max: 50)
    |> foreign_key_constraint(:project_id)
  end
end
