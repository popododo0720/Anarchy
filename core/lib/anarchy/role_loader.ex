defmodule Anarchy.RoleLoader do
  @moduledoc """
  Loads role-specific system prompts from the agency-agents directory.

  Roles map to runtime and model:
  - plan_reviewer, code_reviewer → Codex runtime
  - architect → Claude Code with Opus model
  - All others → Claude Code with Sonnet model
  """

  require Logger

  @agents_path "priv/agency-agents"

  @role_files %{
    architect: "engineering/architect.md",
    senior_developer: "engineering/senior-developer.md",
    developer: "engineering/senior-developer.md",
    qa_engineer: "engineering/qa-engineer.md",
    code_reviewer: "engineering/code-reviewer.md",
    plan_reviewer: "engineering/code-reviewer.md",
    ce_reviewer: "engineering/code-reviewer.md",
    project_manager: "management/project-manager.md",
    pm: "management/project-manager.md"
  }

  @codex_roles ~w(plan_reviewer code_reviewer)a

  @spec load(atom() | String.t()) :: {:ok, String.t()} | {:error, term()}
  def load(role) when is_atom(role) do
    case role_path(role) do
      nil ->
        {:error, {:unknown_role, role}}

      path ->
        case File.read(path) do
          {:ok, content} -> {:ok, content}
          {:error, reason} -> {:error, {:role_file_read_error, path, reason}}
        end
    end
  end

  def load(role) when is_binary(role) do
    load(String.to_existing_atom(role))
  rescue
    ArgumentError -> {:error, {:unknown_role, role}}
  end

  @spec load!(atom() | String.t()) :: String.t()
  def load!(role) do
    case load(role) do
      {:ok, content} -> content
      {:error, reason} -> raise ArgumentError, "Failed to load role #{inspect(role)}: #{inspect(reason)}"
    end
  end

  @spec role_path(atom()) :: Path.t() | nil
  def role_path(role) when is_atom(role) do
    case Map.get(@role_files, role) do
      nil -> nil
      relative -> resolve_agents_path(relative)
    end
  end

  def role_path(_role), do: nil

  @spec runtime_for(atom()) :: module()
  def runtime_for(role) when role in @codex_roles, do: Anarchy.Codex.AppServer
  def runtime_for(_role), do: Anarchy.Runtime.ClaudeCode

  @spec runtime_name(atom()) :: String.t()
  def runtime_name(role) when role in @codex_roles, do: "codex"
  def runtime_name(_role), do: "claude_code"

  @spec model_for(atom()) :: String.t()
  def model_for(:architect), do: "opus"
  def model_for(_role), do: "sonnet"

  @spec execute_role(atom(), map(), Path.t() | nil, String.t()) :: term()
  def execute_role(role, task, workspace_path, prompt) do
    if role in @codex_roles do
      execute_codex_role(task, workspace_path, prompt)
    else
      system_prompt =
        case load(role) do
          {:ok, content} -> content
          {:error, _reason} -> nil
        end

      execute_claude_code_role(role, task, workspace_path, prompt, system_prompt)
    end
  end

  @spec available_roles() :: [atom()]
  def available_roles do
    @role_files
    |> Enum.filter(fn {_role, relative} ->
      path = resolve_agents_path(relative)
      path && File.exists?(path)
    end)
    |> Enum.map(fn {role, _} -> role end)
  end

  @spec list_custom_roles() :: [%{name: String.t(), path: Path.t()}]
  def list_custom_roles do
    custom_dir = resolve_agents_path("custom")

    if custom_dir && File.dir?(custom_dir) do
      custom_dir
      |> File.ls!()
      |> Enum.filter(&String.ends_with?(&1, ".md"))
      |> Enum.map(fn file ->
        %{
          name: Path.rootname(file),
          path: Path.join(custom_dir, file)
        }
      end)
    else
      []
    end
  end

  # --- Private ---

  defp resolve_agents_path(relative) do
    # Check priv/ directory first, then fallback to project root
    priv_path = Path.join(:code.priv_dir(:anarchy) |> to_string(), "agency-agents/#{relative}")

    cond do
      File.exists?(priv_path) ->
        priv_path

      File.exists?(Path.join(@agents_path, relative)) ->
        Path.join(@agents_path, relative)

      true ->
        nil
    end
  rescue
    _ -> nil
  end

  defp execute_codex_role(_task, workspace_path, prompt) do
    case Anarchy.Codex.AppServer.run(workspace_path || System.tmp_dir!(), prompt, %{}) do
      {:ok, result} -> result
      {:error, reason} -> raise "Codex execution failed: #{inspect(reason)}"
    end
  end

  defp execute_claude_code_role(role, _task, workspace_path, prompt, system_prompt) do
    Anarchy.Runtime.ClaudeCode.run_once(
      prompt: prompt,
      model: model_for(role),
      system_prompt: system_prompt,
      workspace_path: workspace_path || System.tmp_dir!()
    )
  end
end
