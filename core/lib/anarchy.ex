defmodule Anarchy do
  @moduledoc """
  Entry point for the Anarchy orchestrator.
  """

  @doc """
  Start the orchestrator in the current BEAM node.
  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    Anarchy.Orchestrator.start_link(opts)
  end
end

defmodule Anarchy.Application do
  @moduledoc """
  OTP application entrypoint that starts core supervisors and workers.
  """

  use Application

  @impl true
  def start(_type, _args) do
    :ok = Anarchy.LogFile.configure()

    children =
      [
        Anarchy.Repo,
        {Phoenix.PubSub, name: Anarchy.PubSub},
        {Task.Supervisor, name: Anarchy.TaskSupervisor},
        {Oban, oban_config()},
        Anarchy.WorkflowStore,
        Anarchy.SessionManager
      ] ++ endpoint_children() ++ runtime_children()

    Supervisor.start_link(
      children,
      strategy: :one_for_one,
      name: Anarchy.Supervisor
    )
  end

  defp endpoint_children do
    if Application.get_env(:anarchy, :env) == :test do
      []
    else
      [AnarchyWeb.Endpoint]
    end
  end

  defp runtime_children do
    if Application.get_env(:anarchy, :env) == :test do
      []
    else
      [
        Anarchy.Orchestrator,
        Anarchy.HttpServer
      ] ++ dashboard_children()
    end
  end

  defp dashboard_children do
    if tui_enabled?(), do: [Anarchy.StatusDashboard], else: []
  end

  @impl true
  def stop(_state) do
    if tui_enabled?(), do: Anarchy.StatusDashboard.render_offline_status()
    :ok
  end

  defp tui_enabled?, do: System.get_env("ANARCHY_TUI") == "1"

  defp oban_config do
    Application.get_env(:anarchy, Oban, [])
  end
end
