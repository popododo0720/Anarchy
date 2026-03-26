# Anarchy API and Communication Rules

Status: mandatory before coding

## Communication model
User/automation -> CLI -> external API
- protocol: HTTP/JSON
- versioned from day one

Internal service/module boundaries
- protocol: gRPC + protobuf by default
- protobuf is the source of truth for internal contracts

External integrations
- Kubernetes API
- Keycloak/OIDC
- image sources
- storage providers
These may use their own protocols at the adapter boundary.

## Rules
- No ad hoc internal JSON buses
- No internal protocol sprawl
- No hidden contracts defined only in implementation

## Required artifacts before implementation
For each externally visible feature:
- external API spec
- CLI command spec
- validation/error semantics

For each internal boundary:
- `.proto` contract
- request/response shape
- error semantics

## Versioning
- external API: `/api/v1/...`
- protobuf packages: `anarchy.<domain>.v1`
