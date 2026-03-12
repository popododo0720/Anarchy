defmodule Anarchy.DataCase do
  @moduledoc """
  Test case for tests that require database access.
  Sets up Ecto.Adapters.SQL.Sandbox for each test.
  """

  use ExUnit.CaseTemplate

  using do
    quote do
      alias Anarchy.Repo
      import Ecto
      import Ecto.Changeset
      import Ecto.Query
    end
  end

  setup tags do
    pid = Ecto.Adapters.SQL.Sandbox.start_owner!(Anarchy.Repo, shared: not tags[:async])

    on_exit(fn -> Ecto.Adapters.SQL.Sandbox.stop_owner(pid) end)

    :ok
  end
end
