defmodule Anarchy.WorkspaceManager do
  @moduledoc """
  Git worktree-based workspace management for agent isolation.

  Each agent gets its own git worktree with a dedicated branch,
  enabling parallel work without conflicts.

  Branch naming: anarchy/{project_id}/{task_id}
  Path: {workspace_root}/{project_id}/{task_id}
  """

  require Logger

  alias Anarchy.Config

  @spec create(String.t(), String.t(), String.t(), String.t()) ::
          {:ok, %{path: Path.t(), branch: String.t()}} | {:error, term()}
  def create(project_id, task_id, repo_path, base_branch \\ "main") do
    path = workspace_path(project_id, task_id)
    branch = branch_name(project_id, task_id)

    with :ok <- ensure_directory(Path.dirname(path)),
         :ok <- create_worktree(repo_path, path, branch, base_branch) do
      Logger.info("Created git worktree: path=#{path} branch=#{branch} base=#{base_branch}")
      {:ok, %{path: path, branch: branch}}
    else
      {:error, reason} ->
        Logger.error("Failed to create worktree: project=#{project_id} task=#{task_id} reason=#{inspect(reason)}")
        {:error, reason}
    end
  end

  @spec remove(String.t(), String.t(), String.t()) :: :ok | {:error, term()}
  def remove(project_id, task_id, repo_path) do
    path = workspace_path(project_id, task_id)

    case remove_worktree(repo_path, path) do
      :ok ->
        Logger.info("Removed git worktree: path=#{path}")
        cleanup_empty_project_dir(project_id)
        :ok

      {:error, reason} ->
        Logger.warning("Failed to remove worktree: path=#{path} reason=#{inspect(reason)}")
        {:error, reason}
    end
  end

  @spec workspace_path(String.t(), String.t()) :: Path.t()
  def workspace_path(project_id, task_id) do
    safe_project = sanitize_id(project_id)
    safe_task = sanitize_id(task_id)
    Path.join([workspace_root(), safe_project, safe_task])
  end

  @spec branch_name(String.t(), String.t()) :: String.t()
  def branch_name(project_id, task_id) do
    safe_project = sanitize_id(project_id)
    safe_task = sanitize_id(task_id)
    "anarchy/#{safe_project}/#{safe_task}"
  end

  @spec list_worktrees(String.t()) :: {:ok, [map()]} | {:error, term()}
  def list_worktrees(repo_path) do
    case System.cmd("git", ["worktree", "list", "--porcelain"], cd: repo_path, stderr_to_stdout: true) do
      {output, 0} ->
        {:ok, parse_worktree_list(output)}

      {output, _code} ->
        {:error, {:git_error, output}}
    end
  rescue
    error -> {:error, {:system_error, error}}
  end

  @spec cleanup_project(String.t(), String.t()) :: :ok
  def cleanup_project(project_id, repo_path) do
    safe_project = sanitize_id(project_id)
    project_dir = Path.join(workspace_root(), safe_project)

    if File.dir?(project_dir) do
      project_dir
      |> File.ls!()
      |> Enum.each(fn task_dir ->
        full_path = Path.join(project_dir, task_dir)

        if File.dir?(full_path) do
          remove_worktree(repo_path, full_path)
        end
      end)

      File.rm_rf(project_dir)
    end

    :ok
  rescue
    _ -> :ok
  end

  @spec exists?(String.t(), String.t()) :: boolean()
  def exists?(project_id, task_id) do
    path = workspace_path(project_id, task_id)
    File.dir?(path)
  end

  # --- Private ---

  defp workspace_root do
    Config.settings!().workspace.root
  rescue
    _ -> Path.join(System.tmp_dir!(), "anarchy_workspaces")
  end

  defp create_worktree(repo_path, path, branch, base_branch) do
    args = ["worktree", "add", "-b", branch, path, base_branch]

    case System.cmd("git", args, cd: repo_path, stderr_to_stdout: true) do
      {_output, 0} ->
        :ok

      {output, _code} ->
        if String.contains?(output, "already exists") do
          # Branch already exists — try to add without -b
          case System.cmd("git", ["worktree", "add", path, branch], cd: repo_path, stderr_to_stdout: true) do
            {_output2, 0} -> :ok
            {output2, _code2} -> {:error, {:git_worktree_add_failed, output2}}
          end
        else
          {:error, {:git_worktree_add_failed, output}}
        end
    end
  rescue
    error -> {:error, {:system_error, error}}
  end

  defp remove_worktree(repo_path, path) do
    case System.cmd("git", ["worktree", "remove", "--force", path], cd: repo_path, stderr_to_stdout: true) do
      {_output, 0} -> :ok
      {_output, _code} -> force_remove_worktree(path)
    end
  rescue
    _ -> force_remove_worktree(path)
  end

  defp force_remove_worktree(path) do
    if File.dir?(path) do
      case File.rm_rf(path) do
        {:ok, _} -> :ok
        {:error, reason, _} -> {:error, {:rm_rf_failed, reason}}
      end
    else
      :ok
    end
  end

  defp ensure_directory(path) do
    case File.mkdir_p(path) do
      :ok -> :ok
      {:error, reason} -> {:error, {:mkdir_failed, path, reason}}
    end
  end

  defp cleanup_empty_project_dir(project_id) do
    safe_project = sanitize_id(project_id)
    project_dir = Path.join(workspace_root(), safe_project)

    if File.dir?(project_dir) do
      case File.ls(project_dir) do
        {:ok, []} -> File.rmdir(project_dir)
        _ -> :ok
      end
    end
  rescue
    _ -> :ok
  end

  defp sanitize_id(id) when is_binary(id) do
    String.replace(id, ~r/[^a-zA-Z0-9._-]/, "_")
  end

  defp parse_worktree_list(output) do
    output
    |> String.split("\n\n", trim: true)
    |> Enum.map(fn block ->
      block
      |> String.split("\n", trim: true)
      |> Enum.reduce(%{}, fn line, acc ->
        case String.split(line, " ", parts: 2) do
          ["worktree", path] -> Map.put(acc, :path, path)
          ["HEAD", sha] -> Map.put(acc, :head, sha)
          ["branch", ref] -> Map.put(acc, :branch, ref)
          ["bare"] -> Map.put(acc, :bare, true)
          ["detached"] -> Map.put(acc, :detached, true)
          _ -> acc
        end
      end)
    end)
    |> Enum.reject(&(&1 == %{}))
  end
end
