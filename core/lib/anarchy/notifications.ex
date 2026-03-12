defmodule Anarchy.Notifications do
  @moduledoc "Real-time notification system for the owner."

  @notify_events [
    :task_completed,
    :critical_found,
    :approval_needed,
    :agent_failed,
    :project_completed,
    :merge_conflict
  ]

  @spec notify(atom(), map()) :: :ok
  def notify(event, payload) when event in @notify_events do
    notification = %{
      id: System.unique_integer([:positive]),
      event: event,
      message: format_message(event, payload),
      severity: severity(event),
      timestamp: DateTime.utc_now(),
      read: false
    }

    Phoenix.PubSub.broadcast(Anarchy.PubSub, "notifications", {:notification, notification})
    :ok
  end

  def notify(_event, _payload), do: :ok

  @spec format_message(atom(), map()) :: String.t()
  def format_message(:critical_found, %{task: t, count: n}),
    do: "#{t.title}: Critical #{n} found"
  def format_message(:task_completed, %{task: t}),
    do: "#{t.title} completed"
  def format_message(:approval_needed, %{task: t}),
    do: "#{t.title} needs approval"
  def format_message(:agent_failed, %{task: t, reason: r}),
    do: "#{t.title} failed: #{inspect(r)}"
  def format_message(:merge_conflict, %{branch: b}),
    do: "Merge conflict: #{b}"
  def format_message(:project_completed, %{project: p}),
    do: "#{p.name} project completed!"
  def format_message(_event, _payload), do: "Unknown notification"

  @spec severity(atom()) :: :high | :medium | :low
  def severity(:critical_found), do: :high
  def severity(:agent_failed), do: :high
  def severity(:merge_conflict), do: :high
  def severity(:approval_needed), do: :medium
  def severity(_), do: :low
end
