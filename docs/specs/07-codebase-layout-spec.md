# Anarchy Codebase Layout Specification

Status: implementation draft

## Structure
- `cmd/anarchy/` for CLI entrypoint
- `cmd/anarchy-api/` for API server entrypoint
- `internal/domain/` for entities and invariants
- `internal/application/` for use cases
- `internal/ports/` for interfaces
- `internal/adapters/` for concrete integrations
- `internal/transport/http/` for external API handlers
- `internal/transport/grpc/` for internal gRPC handlers
- `internal/cli/` for command implementations
- `proto/` for internal contracts
- `charts/anarchy/` for Helm packaging
- `deploy/gitops/` for environment declarations
- `tests/` for automated tests

## Rules
- CLI never directly manipulates Kubernetes
- transport layers never contain domain logic
- adapters never define product semantics
- domain layer never imports infrastructure concerns
