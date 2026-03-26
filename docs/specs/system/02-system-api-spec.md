# Anarchy System API Specification

Status: implementation-ready draft
Version: v1

## GET /api/v1/system/health
Purpose:
- provide platform health summary for CLI and future UI

Response 200 shape:
- `status`
- `apiReachable`
- `kubernetesReachable`
- `kubevirtInstalled`
- `kubevirtReady`
- `cdiInstalled`
- `cdiReady`
- `totalNodes`
- `readyNodes`
- `warnings`

Behavior:
- prefer structured degraded output over hard failure when possible

## GET /api/v1/system/version
Response 200 shape:
- `apiVersion`
- `serverVersion`
- `supportedApiVersions`
- `kubernetesVersion`
- `kubevirtVersion`

## GET /api/v1/system/capabilities
Response 200 shape:
- `vmLifecycleSupported`
- `imageInventorySupported`
- `diagnosticsSupported`
- `publicIpSupported`
- `capabilities`

## Error behavior
If the server cannot complete the request at all:
- return standard error shape
- include machine-readable `code`
- include user-friendly `message`
