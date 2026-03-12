defmodule Anarchy.PMAgent do
  @moduledoc """
  PM (Project Manager) agent that decomposes confirmed designs into tasks.

  Given a design document, analyzes scope, determines PM structure,
  and creates tasks in the database.
  """

  require Logger

  alias Anarchy.{Projects, RoleLoader}
  alias Anarchy.Schemas.Design

  @spec decompose(%Design{}) :: {:ok, [map()]} | {:error, term()}
  def decompose(%Design{status: "confirmed"} = design) do
    Logger.info("PM agent decomposing design_id=#{design.id} title=#{design.title}")

    case run_decomposition(design) do
      {:ok, task_specs} ->
        tasks = create_tasks_from_specs(design, task_specs)
        Logger.info("PM agent created #{length(tasks)} tasks from design_id=#{design.id}")
        {:ok, tasks}

      {:error, reason} ->
        Logger.error("PM agent failed to decompose design_id=#{design.id}: #{inspect(reason)}")
        {:error, reason}
    end
  end

  def decompose(%Design{status: status}) do
    {:error, {:design_not_confirmed, status}}
  end

  @spec decompose_with_agent(%Design{}) :: {:ok, [map()]} | {:error, term()}
  def decompose_with_agent(%Design{status: "confirmed"} = design) do
    prompt = build_decomposition_prompt(design)

    try do
      result = RoleLoader.execute_role(:pm, %{}, nil, prompt)
      task_specs = parse_agent_output(result)
      tasks = create_tasks_from_specs(design, task_specs)
      {:ok, tasks}
    rescue
      error ->
        Logger.error("PM agent runtime error: #{Exception.message(error)}")
        {:error, {:agent_error, Exception.message(error)}}
    end
  end

  def decompose_with_agent(%Design{status: status}) do
    {:error, {:design_not_confirmed, status}}
  end

  @spec estimate_scope(String.t()) :: :small | :medium | :large
  def estimate_scope(content) when is_binary(content) do
    word_count = content |> String.split() |> length()

    cond do
      word_count < 500 -> :small
      word_count < 2000 -> :medium
      true -> :large
    end
  end

  # --- Private ---

  defp run_decomposition(design) do
    scope = estimate_scope(design.content_md)
    task_specs = generate_default_tasks(design, scope)
    {:ok, task_specs}
  end

  defp generate_default_tasks(design, :small) do
    [
      %{title: "Plan: #{design.title}", role: "developer", priority: 1, description: "Create implementation plan for: #{design.title}"},
      %{title: "Implement: #{design.title}", role: "developer", priority: 2, description: "Implement the design: #{design.title}\n\n#{String.slice(design.content_md, 0, 500)}"},
      %{title: "Review: #{design.title}", role: "code_reviewer", priority: 3, description: "Review implementation of: #{design.title}"}
    ]
  end

  defp generate_default_tasks(design, :medium) do
    [
      %{title: "Architecture: #{design.title}", role: "architect", priority: 1, description: "Define architecture for: #{design.title}"},
      %{title: "Plan: #{design.title}", role: "developer", priority: 2, description: "Create detailed implementation plan"},
      %{title: "Core Implementation: #{design.title}", role: "developer", priority: 3, description: "Implement core functionality"},
      %{title: "Integration: #{design.title}", role: "developer", priority: 4, description: "Integration and wiring"},
      %{title: "Testing: #{design.title}", role: "developer", priority: 5, description: "Write tests"},
      %{title: "Review: #{design.title}", role: "code_reviewer", priority: 6, description: "Final code review"}
    ]
  end

  defp generate_default_tasks(design, :large) do
    [
      %{title: "Architecture: #{design.title}", role: "architect", priority: 1, description: "System architecture and component boundaries"},
      %{title: "Sub-decompose: #{design.title}", role: "pm", priority: 2, description: "Break into smaller sub-tasks with dependencies"},
      %{title: "Core Module 1: #{design.title}", role: "developer", priority: 3, description: "First core module implementation"},
      %{title: "Core Module 2: #{design.title}", role: "developer", priority: 3, description: "Second core module implementation"},
      %{title: "Integration Layer: #{design.title}", role: "developer", priority: 4, description: "Connect modules and integration"},
      %{title: "Test Suite: #{design.title}", role: "developer", priority: 5, description: "Comprehensive test coverage"},
      %{title: "CE Review: #{design.title}", role: "ce_reviewer", priority: 6, description: "Security, performance, architecture review"},
      %{title: "Final Review: #{design.title}", role: "code_reviewer", priority: 7, description: "Final code review before merge"}
    ]
  end

  defp create_tasks_from_specs(design, task_specs) do
    task_specs
    |> Enum.with_index()
    |> Enum.map(fn {spec, _idx} ->
      attrs = Map.merge(spec, %{
        project_id: design.project_id,
        design_id: design.id,
        status: :pending
      })

      case Projects.create_task(attrs) do
        {:ok, task} -> task
        {:error, changeset} ->
          Logger.warning("Failed to create task: #{inspect(changeset.errors)}")
          nil
      end
    end)
    |> Enum.reject(&is_nil/1)
  end

  defp build_decomposition_prompt(design) do
    """
    Decompose this design into implementable tasks.

    Design Title: #{design.title}
    Design Content:
    #{design.content_md}

    For each task, output in this format:
    TASK: <title>
    ROLE: <developer|architect|code_reviewer|qa_engineer>
    PRIORITY: <1-10>
    DESCRIPTION: <what to do>
    DEPENDS_ON: <comma-separated task titles, or "none">
    ---
    """
  end

  defp parse_agent_output(output) when is_binary(output) do
    output
    |> String.split("---", trim: true)
    |> Enum.map(&parse_task_block/1)
    |> Enum.reject(&is_nil/1)
  end

  defp parse_agent_output(_output), do: []

  defp parse_task_block(block) do
    lines = String.split(block, "\n", trim: true)

    Enum.reduce(lines, %{}, fn line, acc ->
      cond do
        String.starts_with?(line, "TASK:") ->
          Map.put(acc, :title, String.trim(String.trim_leading(line, "TASK:")))

        String.starts_with?(line, "ROLE:") ->
          Map.put(acc, :role, String.trim(String.trim_leading(line, "ROLE:")))

        String.starts_with?(line, "PRIORITY:") ->
          priority = line |> String.trim_leading("PRIORITY:") |> String.trim() |> String.to_integer()
          Map.put(acc, :priority, priority)

        String.starts_with?(line, "DESCRIPTION:") ->
          Map.put(acc, :description, String.trim(String.trim_leading(line, "DESCRIPTION:")))

        true ->
          acc
      end
    end)
    |> case do
      %{title: _} = spec -> spec
      _ -> nil
    end
  rescue
    _ -> nil
  end
end
