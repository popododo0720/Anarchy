defmodule Anarchy.SessionManager do
  use GenServer

  alias Anarchy.Repo
  alias Anarchy.Schemas.AgentSession
  import Ecto.Query

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  # Public API

  def create_session(attrs) do
    %AgentSession{}
    |> AgentSession.changeset(Map.put(attrs, :started_at, DateTime.utc_now()))
    |> Repo.insert()
  end

  def get_session(session_id) do
    Repo.get_by(AgentSession, session_id: session_id)
  end

  def update_session(session_id, attrs) do
    case get_session(session_id) do
      nil -> {:error, :not_found}
      session -> session |> AgentSession.changeset(attrs) |> Repo.update()
    end
  end

  def pause_session(session_id, reason, context \\ nil) do
    update_session(session_id, %{
      status: "paused",
      pause_reason: reason,
      paused_at: DateTime.utc_now(),
      resume_context: context
    })
  end

  def complete_session(session_id) do
    update_session(session_id, %{
      status: "completed",
      ended_at: DateTime.utc_now()
    })
  end

  def fail_session(session_id) do
    update_session(session_id, %{
      status: "failed",
      ended_at: DateTime.utc_now()
    })
  end

  def active_sessions_for_project(project_id) do
    from(s in AgentSession,
      where: s.project_id == ^project_id and s.status == "active"
    )
    |> Repo.all()
  end

  def touch_session(session_id) do
    update_session(session_id, %{last_active_at: DateTime.utc_now()})
  end

  # GenServer callbacks
  @impl true
  def init(_opts), do: {:ok, %{}}
end
