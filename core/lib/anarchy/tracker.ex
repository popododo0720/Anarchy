defmodule Anarchy.Tracker do
  @moduledoc """
  Adapter boundary for task tracker reads and writes.
  """

  @callback fetch_candidate_tasks() :: {:ok, [term()]} | {:error, term()}
  @callback fetch_tasks_by_states([String.t()]) :: {:ok, [term()]} | {:error, term()}
  @callback fetch_task_states_by_ids([String.t()]) :: {:ok, map()} | {:error, term()}
  @callback update_task_state(String.t(), String.t()) :: {:ok, term()} | {:error, term()}
  @callback fetch_task_by_id(String.t()) :: {:ok, term() | nil} | {:error, term()}

  @spec fetch_candidate_tasks() :: {:ok, [term()]} | {:error, term()}
  def fetch_candidate_tasks do
    adapter().fetch_candidate_tasks()
  end

  @spec fetch_tasks_by_states([String.t()]) :: {:ok, [term()]} | {:error, term()}
  def fetch_tasks_by_states(states) do
    adapter().fetch_tasks_by_states(states)
  end

  @spec fetch_task_states_by_ids([String.t()]) :: {:ok, map()} | {:error, term()}
  def fetch_task_states_by_ids(task_ids) do
    adapter().fetch_task_states_by_ids(task_ids)
  end

  @spec update_task_state(String.t(), String.t()) :: {:ok, term()} | {:error, term()}
  def update_task_state(task_id, new_state) do
    adapter().update_task_state(task_id, new_state)
  end

  @spec fetch_task_by_id(String.t()) :: {:ok, term() | nil} | {:error, term()}
  def fetch_task_by_id(task_id) do
    adapter().fetch_task_by_id(task_id)
  end

  @spec adapter() :: module()
  def adapter do
    case Application.get_env(:anarchy, :tracker_adapter) do
      nil ->
        case Anarchy.Config.tracker_kind() do
          "memory" -> Anarchy.Tracker.Memory
          _ -> Anarchy.Tracker.Postgres
        end

      adapter ->
        adapter
    end
  end
end
