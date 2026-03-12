defmodule Anarchy.SessionHistory do
  @moduledoc "Loads and parses Claude Code session JSONL files."

  alias Anarchy.SessionManager

  @spec load(String.t()) :: {:ok, [map()]} | {:error, term()}
  def load(session_id) do
    _session = SessionManager.get_session(session_id)

    case find_session_file(session_id) do
      nil ->
        {:error, :session_file_not_found}

      path ->
        case File.read(path) do
          {:ok, content} ->
            entries =
              content
              |> String.split("\n", trim: true)
              |> Enum.flat_map(fn line ->
                case Jason.decode(line) do
                  {:ok, parsed} -> [parsed]
                  {:error, _} -> []
                end
              end)
              |> Enum.filter(fn e ->
                e["type"] in ["human", "assistant", "tool_use", "tool_result"]
              end)

            {:ok, entries}

          {:error, reason} ->
            {:error, {:file_read_error, reason}}
        end
    end
  end

  @spec find_session_file(String.t()) :: Path.t() | nil
  def find_session_file(session_id) do
    home = System.user_home!()
    Path.wildcard("#{home}/.claude/projects/**/#{session_id}.jsonl")
    |> List.first()
  end
end
