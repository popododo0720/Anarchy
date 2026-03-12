defmodule Anarchy.PMAgentTest do
  use ExUnit.Case, async: true

  alias Anarchy.PMAgent

  describe "estimate_scope/1" do
    test "returns :small for short content" do
      assert PMAgent.estimate_scope("Short design doc.") == :small
    end

    test "returns :medium for medium content" do
      content = String.duplicate("word ", 800)
      assert PMAgent.estimate_scope(content) == :medium
    end

    test "returns :large for long content" do
      content = String.duplicate("word ", 3000)
      assert PMAgent.estimate_scope(content) == :large
    end
  end

  describe "decompose/1" do
    test "rejects non-confirmed designs" do
      design = %Anarchy.Schemas.Design{
        id: Ecto.UUID.generate(),
        title: "Test",
        content_md: "Content",
        status: "draft",
        project_id: Ecto.UUID.generate()
      }

      assert {:error, {:design_not_confirmed, "draft"}} = PMAgent.decompose(design)
    end
  end
end
