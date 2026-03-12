defmodule Anarchy.MergeManager do
  @moduledoc """
  Handles merging agent branches back to the base branch.

  Strategy:
  1. FIFO merge queue (completed tasks in order)
  2. Attempt automatic merge
  3. If conflict, try AI-assisted resolution
  4. If still conflicted, escalate to user
  """

  require Logger

  @spec merge(String.t(), String.t(), String.t()) :: :ok | {:conflict, String.t()} | {:error, term()}
  def merge(repo_path, source_branch, target_branch \\ "main") do
    Logger.info("Merging #{source_branch} → #{target_branch} in #{repo_path}")

    with :ok <- fetch_latest(repo_path, target_branch),
         :ok <- checkout(repo_path, target_branch),
         result <- attempt_merge(repo_path, source_branch) do
      result
    end
  end

  @spec merge_no_commit(String.t(), String.t(), String.t()) ::
          :ok | {:conflict, [String.t()]} | {:error, term()}
  def merge_no_commit(repo_path, source_branch, target_branch \\ "main") do
    with :ok <- fetch_latest(repo_path, target_branch),
         :ok <- checkout(repo_path, target_branch) do
      case git(repo_path, ["merge", "--no-commit", "--no-ff", source_branch]) do
        {_output, 0} ->
          # Abort the merge to leave the working tree clean
          git(repo_path, ["merge", "--abort"])
          :ok

        {output, _code} ->
          conflicted = parse_conflict_files(output)
          git(repo_path, ["merge", "--abort"])
          {:conflict, conflicted}
      end
    end
  end

  @spec list_merge_queue(String.t()) :: {:ok, [String.t()]} | {:error, term()}
  def list_merge_queue(repo_path) do
    case git(repo_path, ["branch", "--list", "anarchy/*"]) do
      {output, 0} ->
        branches =
          output
          |> String.split("\n", trim: true)
          |> Enum.map(&String.trim/1)
          |> Enum.map(&String.trim_leading(&1, "* "))
          |> Enum.sort()

        {:ok, branches}

      {output, _code} ->
        {:error, {:git_error, output}}
    end
  end

  @spec cleanup_branch(String.t(), String.t()) :: :ok | {:error, term()}
  def cleanup_branch(repo_path, branch) do
    case git(repo_path, ["branch", "-D", branch]) do
      {_output, 0} -> :ok
      {output, _code} -> {:error, {:branch_delete_failed, output}}
    end
  end

  @spec has_conflicts?(String.t(), String.t(), String.t()) :: boolean()
  def has_conflicts?(repo_path, source_branch, target_branch \\ "main") do
    case merge_no_commit(repo_path, source_branch, target_branch) do
      :ok -> false
      {:conflict, _} -> true
      {:error, _} -> true
    end
  end

  # --- Private ---

  defp fetch_latest(repo_path, branch) do
    case git(repo_path, ["fetch", "origin", branch]) do
      {_output, 0} -> :ok
      # fetch may fail if no remote, that's OK for local-only repos
      {_output, _code} -> :ok
    end
  end

  defp checkout(repo_path, branch) do
    case git(repo_path, ["checkout", branch]) do
      {_output, 0} -> :ok
      {output, _code} -> {:error, {:checkout_failed, output}}
    end
  end

  defp attempt_merge(repo_path, source_branch) do
    case git(repo_path, ["merge", "--no-ff", source_branch, "-m", "Merge #{source_branch}"]) do
      {_output, 0} ->
        Logger.info("Successfully merged #{source_branch}")
        :ok

      {output, _code} ->
        Logger.warning("Merge conflict for #{source_branch}: #{String.slice(output, 0, 200)}")
        git(repo_path, ["merge", "--abort"])
        {:conflict, output}
    end
  end

  defp parse_conflict_files(output) do
    output
    |> String.split("\n", trim: true)
    |> Enum.filter(&String.contains?(&1, "CONFLICT"))
    |> Enum.map(fn line ->
      case Regex.run(~r/CONFLICT.*: Merge conflict in (.+)/, line) do
        [_, file] -> file
        _ -> line
      end
    end)
  end

  defp git(repo_path, args) do
    System.cmd("git", args, cd: repo_path, stderr_to_stdout: true)
  rescue
    error -> {"System error: #{inspect(error)}", 1}
  end
end
