# Anarchy System Domain Code Layout

Status: implementation-ready draft

Recommended initial paths:
- `proto/anarchy/system/v1/system.proto`
- `internal/domain/system/types.go`
- `internal/application/system/service.go`
- `internal/ports/system/provider.go`
- `internal/adapters/system/...`
- `internal/transport/http/system/handlers.go`
- `internal/transport/grpc/system/server.go`
- `internal/cli/system/commands.go`
- `tests/unit/system/...`
- `tests/integration/system/...`

Responsibility split:
- domain: types and invariants
- application: orchestration for health/version/capability requests
- ports: provider interfaces
- adapters: environment-specific collection
- http/grpc: transport serialization only
- cli: command wiring and output rendering only
