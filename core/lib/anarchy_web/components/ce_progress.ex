defmodule AnarchyWeb.Components.CEProgress do
  @moduledoc "CE loop progress visualization component."

  use Phoenix.Component

  @ce_stages [
    {:planning, "Plan"},
    {:plan_reviewing, "Plan Review"},
    {:working, "Work"},
    {:ce_reviewing, "CE Review"},
    {:code_reviewing, "Code Review"},
    {:compounding, "Compound"}
  ]

  attr :current_state, :atom, required: true

  def ce_progress(assigns) do
    assigns = assign(assigns, :stages, @ce_stages)

    ~H"""
    <div class="ce-progress">
      <%= for {{stage, label}, idx} <- Enum.with_index(@stages) do %>
        <div class={"ce-stage #{stage_class(stage, @current_state)}"}>
          <div class="ce-stage-dot">
            <%= if stage_class(stage, @current_state) == "completed" do %>
              &#10003;
            <% else %>
              <%= idx + 1 %>
            <% end %>
          </div>
          <span class="ce-stage-label"><%= label %></span>
        </div>
        <%= if stage != :compounding do %>
          <div class={"ce-connector #{if stage_complete?(stage, @current_state), do: "completed", else: ""}"}></div>
        <% end %>
      <% end %>
    </div>
    """
  end

  defp stage_class(stage, current) do
    stage_index = Enum.find_index(@ce_stages, fn {s, _} -> s == stage end)
    current_index = Enum.find_index(@ce_stages, fn {s, _} -> s == current end)

    cond do
      current in [:completed, :failed] -> "completed"
      stage == current -> "active"
      current_index && stage_index < current_index -> "completed"
      true -> "pending"
    end
  end

  defp stage_complete?(stage, current) do
    stage_index = Enum.find_index(@ce_stages, fn {s, _} -> s == stage end)
    current_index = Enum.find_index(@ce_stages, fn {s, _} -> s == current end)
    current in [:completed, :failed] || (current_index && stage_index < current_index)
  end
end
