ExUnit.start()

Ecto.Adapters.SQL.Sandbox.mode(Anarchy.Repo, :manual)

Code.require_file("support/snapshot_support.exs", __DIR__)
Code.require_file("support/test_support.exs", __DIR__)
Code.require_file("support/data_case.exs", __DIR__)
