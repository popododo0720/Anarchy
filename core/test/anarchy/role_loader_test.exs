defmodule Anarchy.RoleLoaderTest do
  use ExUnit.Case, async: true

  alias Anarchy.RoleLoader

  describe "load/1" do
    test "loads architect role prompt" do
      assert {:ok, content} = RoleLoader.load(:architect)
      assert content =~ "Architect"
    end

    test "loads senior_developer role prompt" do
      assert {:ok, content} = RoleLoader.load(:senior_developer)
      assert content =~ "Senior Developer"
    end

    test "loads code_reviewer role prompt" do
      assert {:ok, content} = RoleLoader.load(:code_reviewer)
      assert content =~ "Code Reviewer"
    end

    test "loads project_manager role prompt" do
      assert {:ok, content} = RoleLoader.load(:project_manager)
      assert content =~ "Project Manager"
    end

    test "returns error for unknown role" do
      assert {:error, {:unknown_role, :nonexistent}} = RoleLoader.load(:nonexistent)
    end

    test "loads by string role name" do
      assert {:ok, content} = RoleLoader.load("architect")
      assert content =~ "Architect"
    end

    test "returns error for unknown string role" do
      assert {:error, {:unknown_role, "totally_fake_role"}} = RoleLoader.load("totally_fake_role")
    end
  end

  describe "load!/1" do
    test "returns content for valid role" do
      content = RoleLoader.load!(:architect)
      assert is_binary(content)
      assert content =~ "Architect"
    end

    test "raises for unknown role" do
      assert_raise ArgumentError, fn ->
        RoleLoader.load!(:nonexistent)
      end
    end
  end

  describe "role_path/1" do
    test "returns path for known roles" do
      path = RoleLoader.role_path(:architect)
      assert is_binary(path)
      assert String.ends_with?(path, "architect.md")
    end

    test "returns nil for unknown role" do
      assert nil == RoleLoader.role_path(:unknown_role)
    end
  end

  describe "runtime_for/1" do
    test "returns Codex for plan_reviewer" do
      assert RoleLoader.runtime_for(:plan_reviewer) == Anarchy.Codex.AppServer
    end

    test "returns Codex for code_reviewer" do
      assert RoleLoader.runtime_for(:code_reviewer) == Anarchy.Codex.AppServer
    end

    test "returns ClaudeCode for developer" do
      assert RoleLoader.runtime_for(:developer) == Anarchy.Runtime.ClaudeCode
    end

    test "returns ClaudeCode for architect" do
      assert RoleLoader.runtime_for(:architect) == Anarchy.Runtime.ClaudeCode
    end
  end

  describe "runtime_name/1" do
    test "returns 'codex' for reviewer roles" do
      assert RoleLoader.runtime_name(:plan_reviewer) == "codex"
      assert RoleLoader.runtime_name(:code_reviewer) == "codex"
    end

    test "returns 'claude_code' for other roles" do
      assert RoleLoader.runtime_name(:developer) == "claude_code"
      assert RoleLoader.runtime_name(:architect) == "claude_code"
    end
  end

  describe "model_for/1" do
    test "returns opus for architect" do
      assert RoleLoader.model_for(:architect) == "opus"
    end

    test "returns sonnet for other roles" do
      assert RoleLoader.model_for(:developer) == "sonnet"
      assert RoleLoader.model_for(:code_reviewer) == "sonnet"
      assert RoleLoader.model_for(:pm) == "sonnet"
    end
  end

  describe "available_roles/0" do
    test "returns list of roles with existing files" do
      roles = RoleLoader.available_roles()
      assert is_list(roles)
      assert :architect in roles
      assert :senior_developer in roles
      assert :code_reviewer in roles
    end
  end

  describe "list_custom_roles/0" do
    test "returns list (possibly empty)" do
      roles = RoleLoader.list_custom_roles()
      assert is_list(roles)
    end
  end
end
