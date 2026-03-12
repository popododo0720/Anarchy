defmodule Anarchy.Repo do
  use Ecto.Repo,
    otp_app: :anarchy,
    adapter: Ecto.Adapters.Postgres
end
