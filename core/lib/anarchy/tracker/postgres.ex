defmodule Anarchy.Tracker.Postgres do
  @moduledoc "PostgreSQL-backed tracker adapter for Anarchy."
  @behaviour Anarchy.Tracker

  import Ecto.Query
  alias Anarchy.Repo
  alias Anarchy.Schemas.Task

  @impl true
  def fetch_candidate_tasks do
    tasks =
      from(t in Task,
        where: t.status == :pending,
        order_by: [asc: t.priority, asc: t.inserted_at]
      )
      |> Repo.all()

    all_dep_ids =
      tasks
      |> Enum.flat_map(fn t -> t.depends_on || [] end)
      |> Enum.uniq()

    completed_ids =
      if all_dep_ids == [] do
        MapSet.new()
      else
        from(t in Task,
          where: t.id in ^all_dep_ids and t.status == :completed,
          select: t.id
        )
        |> Repo.all()
        |> MapSet.new()
      end

    filtered =
      Enum.filter(tasks, fn t ->
        deps = t.depends_on || []
        deps == [] or Enum.all?(deps, &MapSet.member?(completed_ids, &1))
      end)

    {:ok, filtered}
  end

  @impl true
  def fetch_task_by_id(task_id) do
    {:ok, Repo.get(Task, task_id)}
  end

  @impl true
  def fetch_tasks_by_states(states) when is_list(states) do
    atom_states = states |> Enum.map(&to_existing_atom/1) |> Enum.reject(&is_nil/1)

    if atom_states == [] do
      {:ok, []}
    else
      tasks = from(t in Task, where: t.status in ^atom_states) |> Repo.all()
      {:ok, tasks}
    end
  end

  @impl true
  def fetch_task_states_by_ids(ids) when is_list(ids) do
    tasks = from(t in Task, where: t.id in ^ids, select: {t.id, t.status}) |> Repo.all()
    {:ok, Map.new(tasks)}
  end

  @impl true
  def update_task_state(task_id, new_state) do
    case Repo.get(Task, task_id) do
      nil ->
        {:error, :not_found}

      task ->
        task
        |> Ecto.Changeset.change(status: to_existing_atom(new_state))
        |> Repo.update()
    end
  end

  @valid_statuses ~w(pending assigned planning plan_reviewing working ce_reviewing code_reviewing compounding completed failed)a

  defp to_existing_atom(value) when is_atom(value), do: value

  defp to_existing_atom(value) when is_binary(value) do
    atom = String.downcase(value) |> String.to_atom()
    if atom in @valid_statuses, do: atom, else: nil
  end
end
