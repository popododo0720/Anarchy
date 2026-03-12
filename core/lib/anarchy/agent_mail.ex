defmodule Anarchy.AgentMail do
  @moduledoc "Agent-to-agent message system."

  import Ecto.Query

  alias Anarchy.Repo
  alias Anarchy.Schemas.AgentMessage

  @spec send(map()) :: {:ok, AgentMessage.t()} | {:error, Ecto.Changeset.t()}
  def send(attrs) do
    result =
      %AgentMessage{}
      |> AgentMessage.changeset(attrs)
      |> Repo.insert()

    case result do
      {:ok, msg} ->
        if msg.to_agent do
          Phoenix.PubSub.broadcast(Anarchy.PubSub, "mail:#{msg.to_agent}", {:new_mail, msg})
        end
        Phoenix.PubSub.broadcast(Anarchy.PubSub, "mail:project:#{msg.project_id}", {:new_mail, msg})
        {:ok, msg}

      error ->
        error
    end
  end

  @spec broadcast(Ecto.UUID.t(), String.t(), String.t(), String.t(), keyword()) :: {:ok, AgentMessage.t()} | {:error, Ecto.Changeset.t()}
  def broadcast(project_id, from, subject, body, opts \\ []) do
    __MODULE__.send(%{
      project_id: project_id,
      from_agent: from,
      to_agent: nil,
      subject: subject,
      body: body,
      type: opts[:type] || "status",
      priority: opts[:priority] || "normal",
      payload: opts[:payload]
    })
  end

  @spec inbox(String.t(), keyword()) :: [AgentMessage.t()]
  def inbox(agent_name, opts \\ []) do
    query =
      from(m in AgentMessage,
        where: m.to_agent == ^agent_name or is_nil(m.to_agent),
        order_by: [desc: m.inserted_at]
      )

    query = if opts[:unread_only], do: where(query, [m], is_nil(m.read_at)), else: query
    query = if opts[:project_id], do: where(query, [m], m.project_id == ^opts[:project_id]), else: query
    query = if opts[:limit], do: limit(query, ^opts[:limit]), else: query
    Repo.all(query)
  end

  @spec mark_read(Ecto.UUID.t()) :: {:ok, AgentMessage.t()} | {:error, Ecto.Changeset.t()}
  def mark_read(message_id) do
    Repo.get!(AgentMessage, message_id)
    |> AgentMessage.changeset(%{read_at: DateTime.utc_now()})
    |> Repo.update()
  end

  @spec thread(Ecto.UUID.t()) :: [AgentMessage.t()]
  def thread(thread_id) do
    from(m in AgentMessage,
      where: m.thread_id == ^thread_id,
      order_by: [asc: m.inserted_at]
    )
    |> Repo.all()
  end

  @spec unread_count(String.t()) :: non_neg_integer()
  def unread_count(agent_name) do
    from(m in AgentMessage,
      where: (m.to_agent == ^agent_name or is_nil(m.to_agent)) and is_nil(m.read_at),
      select: count(m.id)
    )
    |> Repo.one()
  end
end
