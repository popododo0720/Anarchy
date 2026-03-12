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

  require Logger

  @impl true
  def start(_type, _args) do
    :ok = Anarchy.LogFile.configure()
    warn_headless_codex_policy()

    children =
      [
        Anarchy.Repo,
        {Phoenix.PubSub, name: Anarchy.PubSub},
        {Task.Supervisor, name: Anarchy.TaskSupervisor},
        {Oban, oban_config()},
        Anarchy.WorkflowStore,
        Anarchy.SessionManager
      ] ++ endpoint_children() ++ runtime_children()

    result =
      Supervisor.start_link(
        children,
        strategy: :one_for_one,
        name: Anarchy.Supervisor
      )

    # Bootstrap default admin account after Repo is up
    case result do
      {:ok, _pid} -> bootstrap_admin()
      _ -> :ok
    end

    result
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

  defp warn_headless_codex_policy do
    unless tui_enabled?() do
      try do
        policy = Anarchy.Config.settings!().codex.approval_policy

        unless headless_safe_policy?(policy) do
          Logger.warning(
            "Codex approval_policy may block in headless mode. " <>
              "Set codex.approval_policy to \"never\" or use a reject-map in WORKFLOW.md for headless operation."
          )
        end
      rescue
        error ->
          Logger.debug("Skipping headless codex warning: #{Exception.message(error)}")
      end
    end
  end

  defp headless_safe_policy?("never"), do: true
  defp headless_safe_policy?(%{"reject" => _}), do: true
  defp headless_safe_policy?(_), do: false

  defp bootstrap_admin do
    try do
      Anarchy.Accounts.ensure_admin!()
    rescue
      error ->
        Logger.warning("Admin bootstrap skipped: #{Exception.message(error)}")
    end
  end

  defp oban_config do
    Application.get_env(:anarchy, Oban, [])
  end
end
