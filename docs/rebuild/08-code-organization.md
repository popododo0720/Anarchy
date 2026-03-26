# Anarchy Code Organization

Status: pre-development baseline

## Repository structure target
- `docs/`
- `proto/`
- `cmd/`
- `internal/domain/`
- `internal/application/`
- `internal/ports/`
- `internal/adapters/`
- `internal/transport/http/`
- `internal/transport/grpc/`
- `internal/cli/`
- `charts/anarchy/`
- `deploy/gitops/`
- `tests/`

## Layer rules
### domain
Pure business concepts and invariants only.
No Kubernetes client code, no CLI formatting, no transport concerns.

### application
Use cases and orchestration of domain behavior.
Depends on ports, not concrete adapters.

### ports
Interfaces defining what the application layer needs.
Examples:
- storage backend
- image source
- workload repository/manager
- node capability provider

### adapters
Concrete implementations for external systems.
Examples:
- Kubernetes adapter
- KubeVirt adapter
- local storage adapter
- NFS/Ceph adapters later
- auth adapters later

### transport
HTTP/gRPC handlers, serialization, and boundary validation.

### cli
User-facing command behavior only.
Calls the external API; does not bypass it.

## Responsibility rule
No package should need to know more than its layer requires.
This is the practical enforcement of SRP and clean architecture in the codebase.
