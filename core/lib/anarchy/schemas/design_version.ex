defmodule Anarchy.Schemas.DesignVersion do
  @moduledoc false

  use Ecto.Schema
  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "design_versions" do
    field :version, :integer
    field :content_md, :string
    field :change_summary, :string
    field :created_by, :string
    field :inserted_at, :utc_datetime

    belongs_to :design, Anarchy.Schemas.Design
  end

  @required [:design_id, :version, :content_md]
  @optional [:change_summary, :created_by, :inserted_at]

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(version, attrs) do
    version
    |> cast(attrs, @required ++ @optional)
    |> validate_required(@required)
    |> validate_number(:version, greater_than: 0)
    |> validate_length(:change_summary, max: 500)
    |> validate_length(:created_by, max: 100)
    |> foreign_key_constraint(:design_id)
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
