defmodule Anarchy.GitDiff do
  @moduledoc "Git diff viewer for agent changes."

  @spec diff_stat(Anarchy.Schemas.Task.t()) :: {:ok, String.t()} | {:error, term()}
  def diff_stat(task) do
    with {:ok, repo_path} <- task_repo_path(task),
         branch <- task.branch || "anarchy/#{task.id}" do
      case System.cmd("git", ["diff", "main...#{branch}", "--stat"], cd: repo_path, stderr_to_stdout: true) do
        {output, 0} -> {:ok, output}
        {error, _} -> {:error, error}
      end
    end
  end

  @spec full_diff(Anarchy.Schemas.Task.t(), String.t() | nil) :: {:ok, String.t()} | {:error, term()}
  def full_diff(task, file_path \\ nil) do
    with {:ok, repo_path} <- task_repo_path(task),
         branch <- task.branch || "anarchy/#{task.id}" do
      args = ["diff", "main...#{branch}"]
      args = if file_path, do: args ++ ["--", file_path], else: args

      case System.cmd("git", args, cd: repo_path, stderr_to_stdout: true) do
        {output, 0} -> {:ok, output}
        {error, _} -> {:error, error}
      end
    end
  end

  @spec changed_files(Anarchy.Schemas.Task.t()) :: {:ok, [String.t()]} | {:error, term()}
  def changed_files(task) do
    with {:ok, repo_path} <- task_repo_path(task),
         branch <- task.branch || "anarchy/#{task.id}" do
      case System.cmd("git", ["diff", "main...#{branch}", "--name-only"], cd: repo_path, stderr_to_stdout: true) do
        {output, 0} -> {:ok, String.split(output, "\n", trim: true)}
        {error, _} -> {:error, error}
      end
    end
  end

  defp task_repo_path(task) do
    case Anarchy.Projects.get_project(task.project_id) do
      nil -> {:error, :project_not_found}
      project ->
        if project.repo_url do
          # Workspace path from the task's workspace
          {:ok, task.branch && Path.join(System.tmp_dir!(), "anarchy_workspaces/#{task.project_id}/#{task.id}") || System.tmp_dir!()}
        else
          {:error, :no_repo_configured}
        end
    end
  end
end
