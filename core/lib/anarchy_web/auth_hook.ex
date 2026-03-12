defmodule AnarchyWeb.AuthHook do
  @moduledoc """
  LiveView on_mount hook that enforces authentication.
  Reads user_id from session, loads user, redirects to login if missing.
  """

  import Phoenix.LiveView
  import Phoenix.Component

  alias Anarchy.Accounts

  def on_mount(:default, _params, session, socket) do
    case session["user_id"] do
      nil ->
        {:halt, redirect(socket, to: "/login")}

      user_id ->
        case Accounts.get_user(user_id) do
          nil ->
            {:halt, redirect(socket, to: "/login")}

          user ->
            {:cont, assign(socket, :current_user, user)}
        end
    end
  end
end
