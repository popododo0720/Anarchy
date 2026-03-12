defmodule Anarchy.Schemas.User do
  @moduledoc false

  use Ecto.Schema
  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}

  schema "users" do
    field :username, :string
    field :password_hash, :string
    field :role, :string, default: "admin"

    # Virtual field — never persisted
    field :password, :string, virtual: true

    timestamps()
  end

  @spec changeset(%__MODULE__{}, map()) :: Ecto.Changeset.t()
  def changeset(user, attrs) do
    user
    |> cast(attrs, [:username, :password, :role])
    |> validate_required([:username, :password])
    |> validate_length(:username, min: 1, max: 100)
    |> validate_length(:password, min: 1, max: 128)
    |> unique_constraint(:username)
    |> hash_password()
  end

  defp hash_password(%Ecto.Changeset{valid?: true, changes: %{password: password}} = changeset) do
    put_change(changeset, :password_hash, Pbkdf2.hash_pwd_salt(password))
  end

  defp hash_password(changeset), do: changeset
end
