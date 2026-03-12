defmodule Anarchy.MixProject do
  use Mix.Project

  def project do
    [
      app: :anarchy,
      version: "0.1.0",
      elixir: "~> 1.19",
      compilers: [:phoenix_live_view] ++ Mix.compilers(),
      start_permanent: Mix.env() == :prod,
      test_coverage: [
        summary: [
          threshold: 100
        ],
        ignore_modules: [
          Anarchy.Config,
          Anarchy.SpecsCheck,
          Anarchy.Orchestrator,
          Anarchy.Orchestrator.State,
          Anarchy.AgentRunner,
          Anarchy.CLI,
          Anarchy.Codex.AppServer,
          Anarchy.HttpServer,
          Anarchy.StatusDashboard,
          Anarchy.LogFile,
          Anarchy.Workspace,
          Anarchy.WorkspaceManager,
          Anarchy.WorkflowEngine,
          Anarchy.WorkflowEngine.Data,
          Anarchy.Workers.CELoopWorker,
          Anarchy.PMAgent,
          Anarchy.MergeManager,
          Anarchy.ErrorHandler,
          AnarchyWeb.DashboardLive,
          AnarchyWeb.ProjectListLive,
          AnarchyWeb.ProjectDetailLive,
          AnarchyWeb.DesignEditorLive,
          AnarchyWeb.ArchitectChatLive,
          AnarchyWeb.TaskDetailLive,
          AnarchyWeb.AgentMonitorLive,
          AnarchyWeb.Endpoint,
          AnarchyWeb.ErrorHTML,
          AnarchyWeb.ErrorJSON,
          AnarchyWeb.Layouts,
          AnarchyWeb.ObservabilityApiController,
          AnarchyWeb.Presenter,
          AnarchyWeb.StaticAssetController,
          AnarchyWeb.StaticAssets,
          AnarchyWeb.Router,
          AnarchyWeb.Router.Helpers,
          AnarchyWeb.LoginLive,
          AnarchyWeb.AuthController,
          AnarchyWeb.AuthHook
        ]
      ],
      test_ignore_filters: [
        "test/support/snapshot_support.exs",
        "test/support/test_support.exs",
        "test/support/data_case.exs"
      ],
      dialyzer: [
        plt_add_apps: [:mix]
      ],
      escript: escript(),
      aliases: aliases(),
      deps: deps()
    ]
  end

  # Run "mix help compile.app" to learn about applications.
  def application do
    [
      mod: {Anarchy.Application, []},
      extra_applications: [:logger]
    ]
  end

  # Run "mix help deps" to learn about dependencies.
  defp deps do
    [
      {:bandit, "~> 1.8"},
      {:floki, ">= 0.30.0", only: :test},
      {:lazy_html, ">= 0.1.0", only: :test},
      {:phoenix, "~> 1.8.0"},
      {:phoenix_html, "~> 4.2"},
      {:phoenix_live_view, "~> 1.1.0"},
      {:req, "~> 0.5"},
      {:jason, "~> 1.4"},
      {:yaml_elixir, "~> 2.12"},
      {:solid, "~> 1.2"},
      {:ecto, "~> 3.13"},
      {:ecto_sql, "~> 3.12"},
      {:postgrex, "~> 0.19"},
      {:gen_state_machine, "~> 3.0"},
      {:oban, "~> 2.18"},
      {:pbkdf2_elixir, "~> 2.0"},
      {:credo, "~> 1.7", only: [:dev, :test], runtime: false},
      {:dialyxir, "~> 1.4", only: [:dev], runtime: false}
    ]
  end

  defp aliases do
    [
      setup: ["deps.get"],
      build: ["escript.build"],
      lint: ["specs.check", "credo --strict"],
      "ecto.setup": ["ecto.create", "ecto.migrate", "run priv/repo/seeds.exs"],
      "ecto.reset": ["ecto.drop", "ecto.setup"]
    ]
  end

  defp escript do
    [
      app: nil,
      main_module: Anarchy.CLI,
      name: "anarchy",
      path: "bin/anarchy"
    ]
  end
end
