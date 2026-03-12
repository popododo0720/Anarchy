defmodule Anarchy.ProjectsTest do
  use Anarchy.DataCase, async: true

  alias Anarchy.Projects
  alias Anarchy.Schemas.{Project, Design, Task}

  describe "projects" do
    test "create_project/1 with valid data" do
      assert {:ok, project} = Projects.create_project(%{name: "Test Project"})
      assert project.name == "Test Project"
      assert project.status == "active"
      assert project.id != nil
    end

    test "create_project/1 requires name" do
      assert {:error, changeset} = Projects.create_project(%{})
      assert %{name: ["can't be blank"]} = errors_on(changeset)
    end

    test "list_projects/0 returns all projects" do
      {:ok, _} = Projects.create_project(%{name: "Project A"})
      {:ok, _} = Projects.create_project(%{name: "Project B"})
      assert length(Projects.list_projects()) >= 2
    end

    test "get_project!/1 returns the project" do
      {:ok, project} = Projects.create_project(%{name: "Findable"})
      found = Projects.get_project!(project.id)
      assert found.id == project.id
    end

    test "update_project/2 updates the project" do
      {:ok, project} = Projects.create_project(%{name: "Old Name"})
      assert {:ok, updated} = Projects.update_project(project, %{name: "New Name"})
      assert updated.name == "New Name"
    end

    test "delete_project/1 removes the project" do
      {:ok, project} = Projects.create_project(%{name: "To Delete"})
      assert {:ok, _} = Projects.delete_project(project)
      assert_raise Ecto.NoResultsError, fn -> Projects.get_project!(project.id) end
    end
  end

  describe "designs" do
    setup do
      {:ok, project} = Projects.create_project(%{name: "Design Project"})
      %{project: project}
    end

    test "create_design/1 with valid data", %{project: project} do
      attrs = %{title: "Design A", content_md: "# Design", project_id: project.id}
      assert {:ok, design} = Projects.create_design(attrs)
      assert design.title == "Design A"
      assert design.status == "draft"
    end

    test "create_design/1 requires title and content_md", %{project: project} do
      assert {:error, changeset} = Projects.create_design(%{project_id: project.id})
      errors = errors_on(changeset)
      assert errors[:title]
      assert errors[:content_md]
    end

    test "confirm_design/1 sets status and confirmed_at", %{project: project} do
      {:ok, design} = Projects.create_design(%{title: "To Confirm", content_md: "# Content", project_id: project.id})
      assert {:ok, confirmed} = Projects.confirm_design(design)
      assert confirmed.status == "confirmed"
      assert confirmed.confirmed_at != nil
    end

    test "list_designs/1 returns project designs", %{project: project} do
      {:ok, _} = Projects.create_design(%{title: "D1", content_md: "c1", project_id: project.id})
      {:ok, _} = Projects.create_design(%{title: "D2", content_md: "c2", project_id: project.id})
      designs = Projects.list_designs(project.id)
      assert length(designs) >= 2
    end
  end

  describe "tasks" do
    setup do
      {:ok, project} = Projects.create_project(%{name: "Task Project"})
      %{project: project}
    end

    test "create_task/1 with valid data", %{project: project} do
      attrs = %{title: "Task 1", role: "developer", project_id: project.id}
      assert {:ok, task} = Projects.create_task(attrs)
      assert task.title == "Task 1"
      assert task.status == :pending
    end

    test "create_task/1 requires title, role, project_id" do
      assert {:error, changeset} = Projects.create_task(%{})
      errors = errors_on(changeset)
      assert errors[:title]
      assert errors[:role]
      assert errors[:project_id]
    end

    test "list_tasks/1 returns project tasks", %{project: project} do
      {:ok, _} = Projects.create_task(%{title: "T1", role: "developer", project_id: project.id})
      {:ok, _} = Projects.create_task(%{title: "T2", role: "architect", project_id: project.id})
      tasks = Projects.list_tasks(project.id)
      assert length(tasks) >= 2
    end

    test "list_tasks_by_status/2 filters by status", %{project: project} do
      {:ok, _} = Projects.create_task(%{title: "Pending", role: "developer", project_id: project.id})
      pending = Projects.list_tasks_by_status(project.id, :pending)
      assert length(pending) >= 1
      assert Enum.all?(pending, &(&1.status == :pending))
    end
  end

  describe "changesets" do
    test "change_project/2 creates a valid changeset" do
      changeset = Projects.change_project(%Project{}, %{name: "Test Project"})
      assert changeset.valid?
    end

    test "change_design/2 creates a valid changeset" do
      changeset =
        Projects.change_design(%Design{}, %{
          title: "Test Design",
          content_md: "# Design",
          project_id: Ecto.UUID.generate()
        })

      assert changeset.valid?
    end

    test "change_task/2 creates a valid changeset" do
      changeset =
        Projects.change_task(%Task{}, %{
          title: "Test Task",
          role: "developer",
          project_id: Ecto.UUID.generate()
        })

      assert changeset.valid?
    end
  end

  defp errors_on(changeset) do
    Ecto.Changeset.traverse_errors(changeset, fn {msg, opts} ->
      Regex.replace(~r"%{(\w+)}", msg, fn _, key ->
        opts |> Keyword.get(String.to_existing_atom(key), key) |> to_string()
      end)
    end)
  end
end
