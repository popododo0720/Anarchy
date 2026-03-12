defmodule Anarchy.Repo.Migrations.CreateAnarchyTables do
  use Ecto.Migration

  def change do
    # projects
    create table(:projects, primary_key: false) do
      add :id, :uuid, primary_key: true, default: fragment("gen_random_uuid()")
      add :name, :string, null: false, size: 255
      add :description, :text
      add :status, :string, default: "active", size: 50
      add :repo_url, :string, size: 500
      add :base_branch, :string, default: "main", size: 255
      add :config, :map, default: %{}
      timestamps()
    end

    # designs
    create table(:designs, primary_key: false) do
      add :id, :uuid, primary_key: true, default: fragment("gen_random_uuid()")
      add :project_id, references(:projects, type: :uuid, on_delete: :delete_all), null: false
      add :title, :string, null: false, size: 500
      add :content_md, :text, null: false
      add :status, :string, default: "draft", size: 50
      add :version, :integer, default: 1
      add :confirmed_at, :utc_datetime
      timestamps()
    end

    # project_assignments
    create table(:project_assignments, primary_key: false) do
      add :id, :uuid, primary_key: true, default: fragment("gen_random_uuid()")
      add :project_id, references(:projects, type: :uuid, on_delete: :delete_all), null: false
      add :role, :string, null: false, size: 50
      add :scope, :string, size: 500
      add :agent_config, :map, default: %{}
      timestamps(updated_at: false)
    end

    # tasks
    create table(:tasks, primary_key: false) do
      add :id, :uuid, primary_key: true, default: fragment("gen_random_uuid()")
      add :project_id, references(:projects, type: :uuid, on_delete: :delete_all), null: false
      add :design_id, references(:designs, type: :uuid)
      add :parent_task_id, references(:tasks, type: :uuid)
      add :pm_assignment_id, references(:project_assignments, type: :uuid)
      add :title, :string, null: false, size: 500
      add :description, :text
      add :role, :string, null: false, size: 50
      add :status, :string, default: "pending", size: 50
      add :priority, :integer, default: 5
      add :depends_on, {:array, :uuid}, default: []
      add :attempt, :integer, default: 0
      add :max_attempts, :integer, default: 3
      add :pr_url, :string, size: 500
      add :branch, :string, size: 255
      add :result, :map
      add :learnings, :text
      timestamps()
    end

    create index(:tasks, [:project_id, :status])
    create index(:tasks, [:depends_on], using: "GIN")

    # agent_sessions
    create table(:agent_sessions, primary_key: false) do
      add :id, :uuid, primary_key: true, default: fragment("gen_random_uuid()")
      add :task_id, references(:tasks, type: :uuid)
      add :project_id, references(:projects, type: :uuid, on_delete: :delete_all), null: false
      add :agent_type, :string, null: false, size: 50
      add :session_id, :string, null: false, size: 255
      add :role_prompt_path, :string, size: 500
      add :workspace_path, :string, size: 500
      add :branch, :string, size: 255
      add :last_commit_sha, :string, size: 40
      add :status, :string, default: "active", size: 50
      add :pause_reason, :string, size: 100
      add :resume_context, :map
      add :started_at, :utc_datetime, null: false
      add :last_active_at, :utc_datetime
      add :paused_at, :utc_datetime
      add :ended_at, :utc_datetime
    end

    create unique_index(:agent_sessions, [:session_id])
    create index(:agent_sessions, [:project_id])
    create index(:agent_sessions, [:status])
  end
end
