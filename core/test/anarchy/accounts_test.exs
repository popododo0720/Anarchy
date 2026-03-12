defmodule Anarchy.AccountsTest do
  use Anarchy.DataCase

  alias Anarchy.Accounts
  alias Anarchy.Schemas.User

  describe "ensure_admin!/0" do
    test "creates admin user when no users exist" do
      # App bootstrap may have created admin already — delete first to test from scratch
      case Accounts.get_user_by_username("admin") do
        %User{} = u -> Anarchy.Repo.delete(u)
        nil -> :ok
      end

      assert :ok = Accounts.ensure_admin!()

      admin = Accounts.get_user_by_username("admin")
      assert admin != nil
      assert admin.username == "admin"
      assert admin.role == "admin"
    end

    test "does not create duplicate admin" do
      Accounts.ensure_admin!()
      Accounts.ensure_admin!()

      admin_count =
        Anarchy.Repo.all(User)
        |> Enum.count(fn u -> u.username == "admin" end)

      assert admin_count == 1
    end
  end

  describe "authenticate/2" do
    setup do
      Accounts.ensure_admin!()
      :ok
    end

    test "succeeds with correct credentials" do
      assert {:ok, %User{username: "admin"}} = Accounts.authenticate("admin", "admin")
    end

    test "fails with wrong password" do
      assert {:error, :invalid_credentials} = Accounts.authenticate("admin", "wrong")
    end

    test "fails with nonexistent user" do
      assert {:error, :invalid_credentials} = Accounts.authenticate("nobody", "admin")
    end
  end

  describe "create_user/1" do
    test "creates a user with hashed password" do
      {:ok, user} = Accounts.create_user(%{username: "test", password: "secret", role: "admin"})
      assert user.password_hash != nil
      assert user.password_hash != "secret"
      assert Pbkdf2.verify_pass("secret", user.password_hash)
    end

    test "rejects duplicate username" do
      Accounts.create_user(%{username: "dupe", password: "pass", role: "admin"})
      assert {:error, changeset} = Accounts.create_user(%{username: "dupe", password: "pass2", role: "admin"})
      assert {"has already been taken", _} = changeset.errors[:username]
    end
  end
end
