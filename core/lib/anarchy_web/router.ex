defmodule AnarchyWeb.Router do
  @moduledoc """
  Router for Anarchy's observability dashboard and API.
  """

  use Phoenix.Router
  import Phoenix.LiveView.Router

  pipeline :browser do
    plug(:fetch_session)
    plug(:fetch_live_flash)
    plug(:put_root_layout, html: {AnarchyWeb.Layouts, :root})
    plug(:protect_from_forgery)
    plug(:put_secure_browser_headers)
  end

  # Static assets — no auth needed
  scope "/", AnarchyWeb do
    get("/dashboard.css", StaticAssetController, :dashboard_css)
    get("/vendor/phoenix_html/phoenix_html.js", StaticAssetController, :phoenix_html_js)
    get("/vendor/phoenix/phoenix.js", StaticAssetController, :phoenix_js)
    get("/vendor/phoenix_live_view/phoenix_live_view.js", StaticAssetController, :phoenix_live_view_js)
  end

  # Public routes — login page + auth actions
  scope "/", AnarchyWeb do
    pipe_through(:browser)

    live_session :public, layout: {AnarchyWeb.Layouts, :root} do
      live("/login", LoginLive, :login)
    end

    post("/auth/login", AuthController, :login)
    get("/auth/logout", AuthController, :logout)
  end

  # Protected routes — require authenticated session
  scope "/", AnarchyWeb do
    pipe_through(:browser)

    live_session :authenticated,
      layout: {AnarchyWeb.Layouts, :app},
      on_mount: [{AnarchyWeb.AuthHook, :default}] do
      live("/", DashboardLive, :index)
      live("/projects", ProjectListLive, :index)
      live("/projects/:id", ProjectDetailLive, :show)
      live("/projects/:project_id/chat", ArchitectChatLive, :chat)
      live("/designs/:id", DesignEditorLive, :edit)
      live("/tasks/:id", TaskDetailLive, :show)
      live("/agents", AgentMonitorLive, :index)
      live("/projects/:project_id/map", AgentMapLive, :map)
      live("/learnings", LearningsLive, :index)
    end
  end

  # API routes — no browser auth (separate concern)
  scope "/", AnarchyWeb do
    get("/api/v1/state", ObservabilityApiController, :state)

    match(:*, "/api/v1/state", ObservabilityApiController, :method_not_allowed)
    post("/api/v1/refresh", ObservabilityApiController, :refresh)
    match(:*, "/api/v1/refresh", ObservabilityApiController, :method_not_allowed)
    get("/api/v1/:issue_identifier", ObservabilityApiController, :issue)
    match(:*, "/api/v1/:issue_identifier", ObservabilityApiController, :method_not_allowed)
    match(:*, "/*path", ObservabilityApiController, :not_found)
  end
end
