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

    if Enum.empty?(task_specs) do
      {:error, :empty_decomposition}
    else
      {:ok, task_specs}
    end
  end

  defp generate_default_tasks(design, :small) do
    plan = "Plan: #{design.title}"
    impl = "Implement: #{design.title}"
    review = "Review: #{design.title}"

    [
      %{title: plan, role: "developer", priority: 1, description: "Create implementation plan for: #{design.title}", depends_on_titles: []},
      %{title: impl, role: "developer", priority: 2, description: "Implement the design: #{design.title}\n\n#{String.slice(design.content_md, 0, 500)}", depends_on_titles: [plan]},
      %{title: review, role: "code_reviewer", priority: 3, description: "Review implementation of: #{design.title}", depends_on_titles: [impl]}
    ]
  end

  defp generate_default_tasks(design, :medium) do
    arch = "Architecture: #{design.title}"
    plan = "Plan: #{design.title}"
    core = "Core Implementation: #{design.title}"
    integ = "Integration: #{design.title}"
    test = "Testing: #{design.title}"
    review = "Review: #{design.title}"

    [
      %{title: arch, role: "architect", priority: 1, description: "Define architecture for: #{design.title}", depends_on_titles: []},
      %{title: plan, role: "developer", priority: 2, description: "Create detailed implementation plan", depends_on_titles: [arch]},
      %{title: core, role: "developer", priority: 3, description: "Implement core functionality", depends_on_titles: [plan]},
      %{title: integ, role: "developer", priority: 4, description: "Integration and wiring", depends_on_titles: [core]},
      %{title: test, role: "developer", priority: 5, description: "Write tests", depends_on_titles: [integ]},
      %{title: review, role: "code_reviewer", priority: 6, description: "Final code review", depends_on_titles: [test]}
    ]
  end

  defp generate_default_tasks(design, :large) do
    arch = "Architecture: #{design.title}"
    sub = "Sub-decompose: #{design.title}"
    mod1 = "Core Module 1: #{design.title}"
    mod2 = "Core Module 2: #{design.title}"
    integ = "Integration Layer: #{design.title}"
    test = "Test Suite: #{design.title}"
    ce = "CE Review: #{design.title}"
    review = "Final Review: #{design.title}"

    [
      %{title: arch, role: "architect", priority: 1, description: "System architecture and component boundaries", depends_on_titles: []},
      %{title: sub, role: "pm", priority: 2, description: "Break into smaller sub-tasks with dependencies", depends_on_titles: [arch]},
      %{title: mod1, role: "developer", priority: 3, description: "First core module implementation", depends_on_titles: [sub]},
      %{title: mod2, role: "developer", priority: 3, description: "Second core module implementation", depends_on_titles: [sub]},
      %{title: integ, role: "developer", priority: 4, description: "Connect modules and integration", depends_on_titles: [mod1, mod2]},
      %{title: test, role: "developer", priority: 5, description: "Comprehensive test coverage", depends_on_titles: [integ]},
      %{title: ce, role: "ce_reviewer", priority: 6, description: "Security, performance, architecture review", depends_on_titles: [test]},
      %{title: review, role: "code_reviewer", priority: 7, description: "Final code review before merge", depends_on_titles: [ce]}
    ]
  end

  defp create_tasks_from_specs(design, task_specs) do
    # First pass: create all tasks without dependencies
    created =
      task_specs
      |> Enum.map(fn spec ->
        attrs =
          spec
          |> Map.drop([:depends_on_titles])
          |> Map.merge(%{
            project_id: design.project_id,
            design_id: design.id,
            status: :pending
          })

        case Projects.create_task(attrs) do
          {:ok, task} -> {spec, task}
          {:error, changeset} ->
            Logger.warning("Failed to create task: #{inspect(changeset.errors)}")
            nil
        end
      end)
      |> Enum.reject(&is_nil/1)

    # Second pass: resolve depends_on_titles → actual task IDs
    title_to_id = Map.new(created, fn {_spec, task} -> {task.title, task.id} end)

    for {spec, task} <- created do
      dep_titles = Map.get(spec, :depends_on_titles, [])

      dep_ids =
        Enum.flat_map(dep_titles, fn title ->
          case Map.get(title_to_id, title) do
            nil -> []
            id -> [id]
          end
        end)

      parent_id = List.first(dep_ids)

      if dep_ids != [] do
        updates = %{depends_on: dep_ids}
        updates = if parent_id, do: Map.put(updates, :parent_task_id, parent_id), else: updates
        Projects.update_task(task, updates)
      end
    end

    Enum.map(created, fn {_spec, task} -> task end)
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

        String.starts_with?(line, "DEPENDS_ON:") ->
          deps_str = String.trim(String.trim_leading(line, "DEPENDS_ON:"))

          deps =
            if deps_str in ["none", "None", ""] do
              []
            else
              deps_str |> String.split(",") |> Enum.map(&String.trim/1) |> Enum.reject(&(&1 == ""))
            end

          Map.put(acc, :depends_on_titles, deps)

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
