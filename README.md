# Anarchy

Anarchy is a Go-based, CLI-first lightweight private cloud platform built on Kubernetes and KubeVirt.

Current implementation baseline:
- Go-only codebase
- CLI-first product surface
- HTTP/JSON external API
- gRPC/protobuf internal contract direction
- Helm packaging
- GitOps-first deployment model
- KubeVirt-based runtime target
- migration-friendly cloud semantics for AWS/VMware-style environments

Current implemented slices:
- `system` domain
  - `anarchy system health`
  - `anarchy system version`
  - `anarchy system capabilities`
  - `/api/v1/system/health`
  - `/api/v1/system/version`
  - `/api/v1/system/capabilities`
- `node` domain
  - `anarchy node list`
  - `anarchy node show <name>`
  - `/api/v1/nodes`
  - `/api/v1/nodes/{name}`
- `image` domain
  - `anarchy image list`
  - `anarchy image show <name>`
  - `/api/v1/images`
  - `/api/v1/images/{name}`
- `system` and `node` now have Kubernetes-backed real adapters via `kubectl`
- `vm` now uses a Kubernetes/KubeVirt-backed real adapter for create/list/show/start/stop/restart/delete
- `vm` MVP domain
  - `anarchy vm create <name> <image> <cpu> <memory> <network>`
  - `anarchy vm list`
  - `anarchy vm show <name>`
  - `anarchy vm start|stop|restart|delete <name>`
  - `/api/v1/vms`
  - `/api/v1/vms/{name}`
  - `/api/v1/vms/{name}/start`
  - `/api/v1/vms/{name}/stop`
  - `/api/v1/vms/{name}/restart`
  - `/api/v1/vms/{name}/delete`
- `image` now uses a Kubernetes/CDI-backed real adapter for DataSource-based list/show
- `vm create` now uses CDI DataSource + DataVolumeTemplate flow validated on the real node1 lab

Repo structure is being built around:
- `cmd/`
- `internal/domain/`
- `internal/application/`
- `internal/ports/`
- `internal/adapters/`
- `internal/transport/`
- `internal/cli/`
- `proto/`
- `charts/anarchy/`
- `deploy/gitops/`
- `docs/`

Development rules:
- spec first
- TDD first
- adapter-based extensibility
- GitOps-friendly operations
