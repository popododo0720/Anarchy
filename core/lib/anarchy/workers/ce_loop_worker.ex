defmodule Anarchy.Workers.CELoopWorker do
  @moduledoc """
  Oban worker that starts a CE loop for a task.

  Enqueues itself when the Orchestrator dispatches a task that should
  go through the full CE workflow (Plan → Plan Review → Work → CE Review
  → Code Review → Compound).
  """

  use Oban.Worker,
    queue: :ce_loops,
    max_attempts: 3,
    unique: [period: 300, fields: [:args], keys: [:task_id]]

  require Logger

  alias Anarchy.{WorkflowEngine, WorkspaceManager}
  alias Anarchy.Schemas.Task, as: TaskSchema

  @impl Oban.Worker
  def perform(%Oban.Job{args: %{"task_id" => task_id} = args}) do
    Logger.info("CELoopWorker starting for task_id=#{task_id}")

    with {:ok, task} <- fetch_task(task_id),
         {:ok, workspace} <- setup_workspace(task, args) do
      case WorkflowEngine.start_link(task: task, workspace_path: workspace.path) do
        {:ok, pid} ->
          monitor_ref = Process.monitor(pid)
          wait_for_completion(pid, monitor_ref, task_id)

        {:error, {:already_started, _pid}} ->
          Logger.info("CE loop already running for task_id=#{task_id}")
          :ok

        {:error, reason} ->
          Logger.error("Failed to start CE loop for task_id=#{task_id}: #{inspect(reason)}")
          {:error, reason}
      end
    end
  end

  @spec enqueue(String.t(), String.t(), keyword()) :: {:ok, Oban.Job.t()} | {:error, term()}
  def enqueue(task_id, project_id, opts \\ []) do
    repo_path = Keyword.get(opts, :repo_path)
    base_branch = Keyword.get(opts, :base_branch, "main")

    %{task_id: task_id, project_id: project_id, repo_path: repo_path, base_branch: base_branch}
    |> __MODULE__.new()
    |> Oban.insert()
  end

  # --- Private ---

  defp fetch_task(task_id) do
    case Anarchy.Tracker.fetch_task_by_id(task_id) do
      {:ok, nil} -> {:error, {:task_not_found, task_id}}
      {:ok, %TaskSchema{} = task} -> {:ok, task}
      {:ok, task} -> {:ok, task}
      {:error, reason} -> {:error, reason}
    end
  end

  defp setup_workspace(task, args) do
    project_id = task.project_id || args["project_id"]
    repo_path = args["repo_path"]
    base_branch = args["base_branch"] || "main"

    if repo_path do
      WorkspaceManager.create(project_id, task.id, repo_path, base_branch)
    else
      # Fall back to directory-based workspace
      case Anarchy.Workspace.create_for_issue(task) do
        {:ok, path} -> {:ok, %{path: path, branch: nil}}
        {:error, reason} -> {:error, reason}
      end
    end
  end

  defp wait_for_completion(pid, monitor_ref, task_id) do
    result =
      receive do
        {:DOWN, ^monitor_ref, :process, ^pid, :normal} ->
          Logger.info("CE loop completed normally for task_id=#{task_id}")
          :ok

        {:DOWN, ^monitor_ref, :process, ^pid, reason} ->
          Logger.error("CE loop crashed for task_id=#{task_id}: #{inspect(reason)}")
          {:error, reason}
      after
        # 4 hour timeout for entire CE loop
        14_400_000 ->
          Process.exit(pid, :kill)
          {:error, :ce_loop_timeout}
      end

    # On failure, ensure the task is marked failed so it doesn't remain in a phantom active state
    if result != :ok do
      Anarchy.Tracker.update_task_state(task_id, "failed")
    end

    result
  end
end
