defmodule Anarchy.Tracker.Memory do
  @moduledoc "In-memory tracker adapter for testing."
  @behaviour Anarchy.Tracker

  @impl true
  def fetch_candidate_tasks do
    issues = Application.get_env(:anarchy, :memory_tracker_issues, [])
    {:ok, issues}
  end

  @impl true
  def fetch_task_by_id(task_id) do
    issues = Application.get_env(:anarchy, :memory_tracker_issues, [])
    {:ok, Enum.find(issues, fn i -> Map.get(i, :id) == task_id end)}
  end

  @impl true
  def fetch_tasks_by_states(states) when is_list(states) do
    issues = Application.get_env(:anarchy, :memory_tracker_issues, [])

    filtered =
      Enum.filter(issues, fn i ->
        state = Map.get(i, :state) || Map.get(i, :status)
        to_string(state) in states
      end)

    {:ok, filtered}
  end

  @impl true
  def fetch_task_states_by_ids(ids) when is_list(ids) do
    issues = Application.get_env(:anarchy, :memory_tracker_issues, [])
    id_set = MapSet.new(ids)

    states =
      issues
      |> Enum.filter(fn i -> MapSet.member?(id_set, Map.get(i, :id)) end)
      |> Map.new(fn i ->
        status = Map.get(i, :status) || Map.get(i, :state)
        {Map.get(i, :id), status}
      end)

    {:ok, states}
  end

  @impl true
  def update_task_state(task_id, new_state) do
    issues = Application.get_env(:anarchy, :memory_tracker_issues, [])

    updated =
      Enum.map(issues, fn i ->
        if Map.get(i, :id) == task_id do
          Map.put(i, :state, new_state)
        else
          i
        end
      end)

    Application.put_env(:anarchy, :memory_tracker_issues, updated)
    {:ok, Enum.find(updated, fn i -> Map.get(i, :id) == task_id end)}
  end
end
