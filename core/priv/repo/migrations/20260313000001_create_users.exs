defmodule Anarchy.Repo.Migrations.CreateUsers do
  use Ecto.Migration

  def change do
    create table(:users, primary_key: false) do
      add :id, :binary_id, primary_key: true
      add :username, :string, null: false
      add :password_hash, :string, null: false
      add :role, :string, null: false, default: "admin"

      timestamps()
    end

    create unique_index(:users, [:username])
  end
end
