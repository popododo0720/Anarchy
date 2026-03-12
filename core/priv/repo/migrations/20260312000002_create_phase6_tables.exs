defmodule Anarchy.Repo.Migrations.CreatePhase6Tables do
  use Ecto.Migration

  def change do
    create table(:agent_messages, primary_key: false) do
      add :id, :uuid, primary_key: true, default: fragment("gen_random_uuid()")
      add :project_id, references(:projects, type: :uuid, on_delete: :delete_all), null: false
      add :from_agent, :string, null: false, size: 255
      add :to_agent, :string, size: 255
      add :thread_id, :uuid
      add :subject, :string, null: false, size: 500
      add :body, :text, null: false
      add :type, :string, null: false, size: 50
      add :priority, :string, default: "normal", size: 20
      add :payload, :map
      add :read_at, :utc_datetime
      add :inserted_at, :utc_datetime, null: false, default: fragment("now()")
    end

    create index(:agent_messages, [:to_agent], where: "read_at IS NULL", name: :idx_messages_to_unread)
    create index(:agent_messages, [:project_id])
    create index(:agent_messages, [:thread_id])

    create table(:design_versions, primary_key: false) do
      add :id, :uuid, primary_key: true, default: fragment("gen_random_uuid()")
      add :design_id, references(:designs, type: :uuid, on_delete: :delete_all), null: false
      add :version, :integer, null: false
      add :content_md, :text, null: false
      add :change_summary, :string, size: 500
      add :created_by, :string, size: 100
      add :inserted_at, :utc_datetime, null: false, default: fragment("now()")
    end

    create index(:design_versions, [:design_id, :version])
  end
end
