defmodule Anarchy.WorkspaceManagerTest do
  use ExUnit.Case, async: true

  alias Anarchy.WorkspaceManager

  describe "workspace_path/2" do
    test "generates path from project and task ids" do
      path = WorkspaceManager.workspace_path("proj-1", "task-1")
      assert is_binary(path)
      assert String.contains?(path, "proj-1")
      assert String.contains?(path, "task-1")
    end

    test "sanitizes unsafe characters in ids" do
      path = WorkspaceManager.workspace_path("proj/1", "task..1")
      assert is_binary(path)
      refute String.contains?(path, "/1")
    end
  end

  describe "branch_name/2" do
    test "generates anarchy-prefixed branch name" do
      branch = WorkspaceManager.branch_name("proj-1", "task-1")
      assert branch == "anarchy/proj-1/task-1"
    end

    test "sanitizes unsafe characters" do
      branch = WorkspaceManager.branch_name("proj/evil", "task<bad>")
      refute String.contains?(branch, "<")
      refute String.contains?(branch, ">")
      assert String.starts_with?(branch, "anarchy/")
    end
  end

  describe "exists?/2" do
    test "returns false for non-existent workspace" do
      refute WorkspaceManager.exists?("nonexistent-proj", "nonexistent-task")
    end
  end
end
