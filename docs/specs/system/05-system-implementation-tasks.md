# Anarchy System Implementation Tasks

Status: implementation-ready draft

## Goal
Deliver the first working domain: `system`

## Task S1 — Create proto definitions
Files expected:
- `proto/anarchy/system/v1/system.proto`
- optional updates under `proto/anarchy/common/v1/`

Done when:
- service and message types exist
- code generation target is known

## Task S2 — Create domain/application contracts
Files expected:
- `internal/domain/system/...`
- `internal/application/system/...`
- `internal/ports/system/...`

Done when:
- health/version/capability result models exist
- required ports are defined cleanly

## Task S3 — Create fakeable system adapter contract
Purpose:
- backend can be replaced in tests without real Kubernetes

Done when:
- adapter-facing interfaces are testable

## Task S4 — Implement HTTP handlers
Files expected:
- `internal/transport/http/system/...`

Done when:
- `/api/v1/system/health`
- `/api/v1/system/version`
- `/api/v1/system/capabilities`
  are served from the application layer

## Task S5 — Implement CLI commands
Files expected:
- `internal/cli/system/...`
- `cmd/anarchy/...`

Done when:
- all three commands call the external API and render output

## Task S6 — Add tests
Files expected under:
- `tests/unit/...`
- `tests/integration/...`

Done when:
- unit, contract, and initial integration tests pass

## Task S7 — Add real Kubernetes/KubeVirt adapter later
Initial implementation may use a simplified adapter shape.
Real environment-backed implementation follows once codebase skeleton and lab are ready.
