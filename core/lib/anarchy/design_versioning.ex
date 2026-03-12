defmodule Anarchy.DesignVersioning do
  @moduledoc "Design document version tracking and diff."

  import Ecto.Query

  alias Anarchy.Repo
  alias Anarchy.Schemas.DesignVersion

  @spec save_version(Anarchy.Schemas.Design.t(), String.t() | nil, String.t()) :: {:ok, DesignVersion.t()} | {:error, Ecto.Changeset.t()}
  def save_version(design, change_summary \\ nil, created_by \\ "user") do
    version =
      (Repo.one(
         from(v in DesignVersion,
           where: v.design_id == ^design.id,
           select: max(v.version)
         )
       ) || 0) + 1

    %DesignVersion{}
    |> DesignVersion.changeset(%{
      design_id: design.id,
      version: version,
      content_md: design.content_md,
      change_summary: change_summary,
      created_by: created_by
    })
    |> Repo.insert()
  end

  @spec diff(Ecto.UUID.t(), integer(), integer()) :: [{:eq | :ins | :del, String.t()}]
  def diff(design_id, version_a, version_b) do
    a = Repo.get_by!(DesignVersion, design_id: design_id, version: version_a)
    b = Repo.get_by!(DesignVersion, design_id: design_id, version: version_b)
    String.myers_difference(a.content_md, b.content_md)
  end

  @spec history(Ecto.UUID.t()) :: [DesignVersion.t()]
  def history(design_id) do
    from(v in DesignVersion,
      where: v.design_id == ^design_id,
      order_by: [desc: v.version]
    )
    |> Repo.all()
  end

  @spec get_version(Ecto.UUID.t(), integer()) :: DesignVersion.t() | nil
  def get_version(design_id, version) do
    Repo.get_by(DesignVersion, design_id: design_id, version: version)
  end
end
