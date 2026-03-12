defmodule Anarchy.Accounts do
  @moduledoc """
  Authentication context — user lookup, password verification, admin bootstrap.
  """

  alias Anarchy.Repo
  alias Anarchy.Schemas.User

  @spec get_user(String.t()) :: User.t() | nil
  def get_user(id), do: Repo.get(User, id)

  @spec get_user_by_username(String.t()) :: User.t() | nil
  def get_user_by_username(username) do
    Repo.get_by(User, username: username)
  end

  @spec verify_password(User.t(), String.t()) :: boolean()
  def verify_password(%User{password_hash: hash}, password) do
    Pbkdf2.verify_pass(password, hash)
  end

  @spec authenticate(String.t(), String.t()) :: {:ok, User.t()} | {:error, :invalid_credentials}
  def authenticate(username, password) do
    case get_user_by_username(username) do
      %User{} = user ->
        if verify_password(user, password) do
          {:ok, user}
        else
          # Constant-time comparison already done by Pbkdf2.verify_pass
          {:error, :invalid_credentials}
        end

      nil ->
        # Prevent timing attack — hash a dummy password even when user doesn't exist
        Pbkdf2.no_user_verify()
        {:error, :invalid_credentials}
    end
  end

  @spec create_user(map()) :: {:ok, User.t()} | {:error, Ecto.Changeset.t()}
  def create_user(attrs) do
    %User{}
    |> User.changeset(attrs)
    |> Repo.insert()
  end

  @doc """
  Ensure a default admin account exists when the users table is empty.
  Safe to call multiple times — no-op if any user exists.
  """
  @spec ensure_admin!() :: :ok
  def ensure_admin! do
    count = Repo.aggregate(User, :count)

    if count == 0 do
      {:ok, _user} =
        create_user(%{
          username: "admin",
          password: "admin",
          role: "admin"
        })
    end

    :ok
  end
end
