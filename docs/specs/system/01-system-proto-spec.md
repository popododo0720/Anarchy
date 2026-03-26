# Anarchy System Proto Specification

Status: implementation-ready draft
Package: `anarchy.system.v1`

## Service
`SystemService`

### RPC: GetHealth
Request:
- empty or metadata-only request

Response fields:
- `status` (healthy/degraded/unhealthy)
- `api_reachable`
- `kubernetes_reachable`
- `kubevirt_installed`
- `kubevirt_ready`
- `cdi_installed`
- `cdi_ready`
- `total_nodes`
- `ready_nodes`
- `warnings[]`

### RPC: GetVersion
Response fields:
- `api_version`
- `server_version`
- `supported_api_versions[]`
- `kubernetes_version`
- `kubevirt_version`

### RPC: GetCapabilities
Response fields:
- `vm_lifecycle_supported`
- `image_inventory_supported`
- `diagnostics_supported`
- `public_ip_supported`
- `capabilities[]`

## Shared types
Use `anarchy.common.v1` for common enums/messages where useful:
- HealthStatus enum
- Warning/Condition-like summary type if reused later

## Versioning rule
Do not add breaking field changes without a new version namespace.
