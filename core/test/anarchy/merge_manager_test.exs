defmodule Anarchy.MergeManagerTest do
  use ExUnit.Case, async: true

  alias Anarchy.MergeManager

  # Git worktree/merge operations require a real git repo.
  # These tests verify the API shape and error handling.

  describe "list_merge_queue/1" do
    test "returns error for non-git directory" do
      result = MergeManager.list_merge_queue(System.tmp_dir!())
      assert {:error, _} = result
    end
  end

  describe "has_conflicts?/3" do
    test "returns true for non-git directory (error case)" do
      assert MergeManager.has_conflicts?(System.tmp_dir!(), "fake-branch")
    end
  end

  describe "cleanup_branch/2" do
    test "returns error for non-git directory" do
      assert {:error, _} = MergeManager.cleanup_branch(System.tmp_dir!(), "fake-branch")
    end
  end
end
