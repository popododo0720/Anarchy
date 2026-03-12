defmodule AnarchyWeb.Endpoint do
  @moduledoc """
  Phoenix endpoint for Anarchy's optional observability UI and API.
  """

  use Phoenix.Endpoint, otp_app: :anarchy

  @session_options [
    store: :cookie,
    key: "_anarchy_key",
    signing_salt: "anarchy-session"
  ]

  socket("/live", Phoenix.LiveView.Socket,
    websocket: [connect_info: [session: @session_options]],
    longpoll: false
  )

  plug(Plug.RequestId)
  plug(Plug.Telemetry, event_prefix: [:phoenix, :endpoint])

  plug(Plug.Parsers,
    parsers: [:urlencoded, :multipart, :json],
    pass: ["*/*"],
    json_decoder: Jason
  )

  plug(Plug.MethodOverride)
  plug(Plug.Head)
  plug(Plug.Session, @session_options)
  plug(AnarchyWeb.Router)
end
