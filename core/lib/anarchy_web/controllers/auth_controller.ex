defmodule AnarchyWeb.AuthController do
  @moduledoc """
  Handles login POST and logout.
  Login sets the session; LiveView reads it via on_mount.
  """

  use Phoenix.Controller, formats: [:html]

  alias Anarchy.Accounts

  @spec login(Plug.Conn.t(), map()) :: Plug.Conn.t()
  def login(conn, %{"username" => username, "password" => password}) do
    # Lazy bootstrap — ensure admin exists on first login attempt
    Accounts.ensure_admin!()

    case Accounts.authenticate(username, password) do
      {:ok, user} ->
        conn
        |> put_session(:user_id, user.id)
        |> configure_session(renew: true)
        |> redirect(to: "/")

      {:error, :invalid_credentials} ->
        conn
        |> put_flash(:error, "Invalid username or password")
        |> redirect(to: "/login")
    end
  end

  @spec logout(Plug.Conn.t(), map()) :: Plug.Conn.t()
  def logout(conn, _params) do
    conn
    |> configure_session(drop: true)
    |> redirect(to: "/login")
  end
end
