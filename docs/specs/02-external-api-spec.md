# Anarchy External API Specification

Status: implementation draft
Version: v1
Protocol: HTTP/JSON

## 1. API principles
- versioned from day one
- stable semantics for CLI and future UI
- explicit request/response structures
- predictable error model

Base path:
- `/api/v1`

## 2. Error response shape
All non-2xx responses should follow a consistent shape:
- `code`: machine-readable error code
- `message`: human-readable summary
- `details`: optional structured detail

## 3. System endpoints
### GET /api/v1/system/health
Returns:
- api status
- kubernetes status
- kubevirt status
- cdi status
- node readiness summary

### GET /api/v1/system/version
Returns:
- apiVersion
- serverVersion
- supportedApiVersions

### GET /api/v1/system/capabilities
Returns:
- list of supported features/capabilities

## 4. Node endpoints
### GET /api/v1/nodes
Returns node summary list

### GET /api/v1/nodes/{name}
Returns node detail

## 5. Image endpoints
### GET /api/v1/images
Returns image summary list

### GET /api/v1/images/{name}
Returns image detail

## 6. VM endpoints
### POST /api/v1/vms
Creates VM
Initial request body fields:
- name
- imageRef
- cpu
- memory
- networkAttachments
- optional volumeRequests
- optional placementHints

### GET /api/v1/vms
Returns VM summary list

### GET /api/v1/vms/{name}
Returns VM detail

### POST /api/v1/vms/{name}:start
Starts VM

### POST /api/v1/vms/{name}:stop
Stops VM

### POST /api/v1/vms/{name}:restart
Restarts VM

### DELETE /api/v1/vms/{name}
Deletes VM

## 7. Diagnose endpoints
### GET /api/v1/diagnostics/cluster
Returns cluster diagnostic summary

### GET /api/v1/diagnostics/vms/{name}
Returns VM diagnostic detail

## 8. Future API domains
Reserved for later:
- network
- subnet
- public-ip
- security-policy
- volume
- auth/project/tenant

These domains should be added without breaking the initial API shape.
