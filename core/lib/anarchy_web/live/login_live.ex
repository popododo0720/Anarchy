defmodule AnarchyWeb.LoginLive do
  @moduledoc """
  Login page — username/password form.
  Authenticates via Accounts context and sets session via a POST redirect.
  """

  use Phoenix.LiveView

  @impl true
  def mount(_params, _session, socket) do
    {:ok, assign(socket, error: nil, page_title: "Login")}
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="login-container">
      <div class="login-card">
        <h1>Anarchy</h1>
        <p class="text-muted">Sign in to continue</p>

        <%= if @error do %>
          <div class="flash flash-error"><%= @error %></div>
        <% end %>

        <form action="/auth/login" method="post" class="login-form">
          <input type="hidden" name="_csrf_token" value={Plug.CSRFProtection.get_csrf_token()} />
          <div class="form-group">
            <label for="username">Username</label>
            <input type="text" id="username" name="username" class="input" required autofocus />
          </div>
          <div class="form-group">
            <label for="password">Password</label>
            <input type="password" id="password" name="password" class="input" required />
          </div>
          <button type="submit" class="btn btn-primary btn-block">Sign In</button>
        </form>
      </div>
    </div>
    """
  end
end
