defmodule Anarchy.WorkflowEngineTest do
  use ExUnit.Case, async: false

  alias Anarchy.WorkflowEngine
  alias Anarchy.Schemas.Task, as: TaskSchema

  setup do
    # Use memory tracker for tests
    Application.put_env(:anarchy, :tracker_adapter, Anarchy.Tracker.Memory)
    Application.put_env(:anarchy, :memory_tracker_issues, [])

    on_exit(fn ->
      Application.delete_env(:anarchy, :tracker_adapter)
      Application.delete_env(:anarchy, :memory_tracker_issues)
    end)
  end

  defp make_task(attrs \\ %{}) do
    defaults = %{
      id: Ecto.UUID.generate(),
      project_id: Ecto.UUID.generate(),
      title: "Test task",
      description: "A test task for CE loop",
      role: "developer",
      status: :pending,
      priority: 5,
      attempt: 0,
      max_attempts: 3,
      depends_on: [],
      inserted_at: NaiveDateTime.utc_now(),
      updated_at: NaiveDateTime.utc_now()
    }

    struct(TaskSchema, Map.merge(defaults, attrs))
  end

  describe "start_link/1" do
    test "starts the workflow engine in :idle state then transitions" do
      task = make_task()

      name = Module.concat(__MODULE__, :"WE_#{System.unique_integer([:positive])}")
      {:ok, pid} = WorkflowEngine.start_link(task: task, name: name)

      assert Process.alive?(pid)

      # Give it a moment to process initial events
      Process.sleep(50)

      {state, data} = WorkflowEngine.current_state(name)
      # It should have transitioned past :idle (to :planning or further)
      # Since RoleLoader.execute_role will fail (no Claude Code installed),
      # it will likely be in :planning or :failed
      assert state in [:planning, :failed]
      assert data.task.id == task.id

      GenStateMachine.stop(pid)
    end
  end

  describe "trigger/2" do
    test "can cancel a running workflow" do
      task = make_task()

      name = Module.concat(__MODULE__, :"WE_Cancel_#{System.unique_integer([:positive])}")
      {:ok, pid} = WorkflowEngine.start_link(task: task, name: name)

      Process.sleep(50)

      WorkflowEngine.trigger(name, :cancel)
      Process.sleep(50)

      {state, _data} = WorkflowEngine.current_state(name)
      assert state == :failed

      GenStateMachine.stop(pid)
    end
  end

  describe "Data struct" do
    test "has correct defaults" do
      data = %WorkflowEngine.Data{}
      assert data.attempt == 0
      assert data.max_attempts == 3
      assert data.ce_review_results == []
      assert data.current_worker_pid == nil
    end
  end
end
