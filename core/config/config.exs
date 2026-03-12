import Config

config :phoenix, :json_library, Jason

config :anarchy, AnarchyWeb.Endpoint,
  adapter: Bandit.PhoenixAdapter,
  url: [host: "localhost"],
  render_errors: [
    formats: [html: AnarchyWeb.ErrorHTML, json: AnarchyWeb.ErrorJSON],
    layout: false
  ],
  pubsub_server: Anarchy.PubSub,
  live_view: [signing_salt: "anarchy-live-view"],
  secret_key_base: String.duplicate("s", 64),
  check_origin: false,
  http: [port: 4000],
  server: true

config :anarchy,
  env: config_env(),
  ecto_repos: [Anarchy.Repo]

# Dev-only DB config. Production uses DATABASE_URL via config/runtime.exs.
if config_env() in [:dev, :test] do
  repo_config = [
    database: "anarchy_#{config_env()}",
    username: "postgres",
    password: "postgres",
    hostname: "localhost",
    port: 5432,
    pool_size: 10
  ]

  repo_config =
    if config_env() == :test do
      Keyword.put(repo_config, :pool, Ecto.Adapters.SQL.Sandbox)
    else
      repo_config
    end

  config :anarchy, Anarchy.Repo, repo_config
end

oban_base = [
  engine: Oban.Engines.Basic,
  repo: Anarchy.Repo,
  queues: [ce_loops: 5, default: 10]
]

if config_env() == :test do
  config :anarchy, Oban, Keyword.put(oban_base, :testing, :manual)
  config :anarchy, AnarchyWeb.Endpoint, server: false
else
  config :anarchy, Oban, oban_base
end
