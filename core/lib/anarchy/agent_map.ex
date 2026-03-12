defmodule Anarchy.AgentMap do
  @moduledoc "Builds agent hierarchy tree for a project."

  alias Anarchy.{Projects, SessionManager, WorkflowEngine}

  @spec build_tree(Ecto.UUID.t()) :: map()
  def build_tree(project_id) do
    assignments = Projects.list_assignments(project_id)
    tasks = Projects.list_tasks(project_id)
    sessions = SessionManager.active_sessions_for_project(project_id)

    %{
      project_id: project_id,
      architect: build_role_node(:architect, tasks, sessions),
      pms:
        Enum.map(assignments, fn a ->
          pm_tasks = Enum.filter(tasks, &(&1.pm_assignment_id == a.id))

          %{
            id: a.id,
            role: a.role,
            scope: a.scope,
            status: infer_status(pm_tasks),
            children:
              Enum.map(pm_tasks, fn t ->
                session = Enum.find(sessions, &(&1.task_id == t.id))

                %{
                  task_id: t.id,
                  title: t.title,
                  role: t.role,
                  status: t.status,
                  ce_state: get_ce_state(t.id),
                  session: format_session(session)
                }
              end)
          }
        end),
      unassigned_tasks:
        tasks
        |> Enum.filter(&is_nil(&1.pm_assignment_id))
        |> Enum.map(fn t ->
          session = Enum.find(sessions, &(&1.task_id == t.id))

          %{
            task_id: t.id,
            title: t.title,
            role: t.role,
            status: t.status,
            ce_state: get_ce_state(t.id),
            session: format_session(session)
          }
        end)
    }
  end

  defp build_role_node(role, tasks, sessions) do
    role_str = Atom.to_string(role)
    role_tasks = Enum.filter(tasks, &(&1.role == role_str))

    %{
      role: role,
      tasks: Enum.map(role_tasks, fn t ->
        session = Enum.find(sessions, &(&1.task_id == t.id))
        %{task_id: t.id, title: t.title, status: t.status, session: format_session(session)}
      end)
    }
  end

  defp format_session(nil), do: nil
  defp format_session(session) do
    %{
      session_id: session.session_id,
      agent_type: session.agent_type,
      status: session.status,
      last_active_at: session.last_active_at
    }
  end

  defp infer_status(tasks) do
    cond do
      Enum.all?(tasks, &(&1.status == :completed)) -> :completed
      Enum.any?(tasks, &(&1.status == :failed)) -> :has_failures
      Enum.any?(tasks, &(&1.status in [:working, :planning, :ce_reviewing, :code_reviewing])) -> :active
      true -> :pending
    end
  end

  defp get_ce_state(task_id) do
    try do
      case WorkflowEngine.current_state({:global, {WorkflowEngine, task_id}}) do
        {state, _data} -> state
      end
    catch
      :exit, _ -> nil
    end
  end
end
