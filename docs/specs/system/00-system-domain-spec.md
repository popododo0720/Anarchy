# Anarchy System Domain Specification

Status: implementation-ready draft
Domain: system

## 1. Purpose
The system domain provides baseline platform awareness.
It answers:
- is the Anarchy API reachable?
- is Kubernetes reachable?
- is KubeVirt installed and ready?
- is CDI installed and ready?
- what versions and capabilities are available?

This is the first implementation domain because every later domain depends on a reliable system baseline.

## 2. Scope
Included in v1:
- platform health summary
- version summary
- capability summary

Not included yet:
- deep node-level diagnostics beyond summary references
- remediation actions
- long historical health reporting

## 3. External CLI surface
Commands:
- `anarchy system health`
- `anarchy system version`
- `anarchy system capabilities`

## 4. External API surface
Endpoints:
- `GET /api/v1/system/health`
- `GET /api/v1/system/version`
- `GET /api/v1/system/capabilities`

## 5. Internal gRPC surface
Service: `anarchy.system.v1.SystemService`
RPCs:
- `GetHealth`
- `GetVersion`
- `GetCapabilities`

## 6. Domain outputs
### Health summary
Must include at minimum:
- api reachable
- kubernetes reachable
- kubevirt installed
- kubevirt ready
- cdi installed
- cdi ready
- total node count
- ready node count
- warnings list

### Version summary
Must include at minimum:
- cli version
- api version
- server version
- supported API versions
- Kubernetes version if available
- KubeVirt version if available

### Capability summary
Must include at minimum:
- vm lifecycle support
- image inventory support
- diagnostics support
- public IP support flag
- storage backend summary if available later

## 7. Error semantics
Health endpoint should prefer degraded status over hard failure when possible.
Examples:
- API alive but KubeVirt missing => health returns 200 with degraded flags
- API cannot talk to Kubernetes => health may still return a structured unhealthy response

Version/capabilities may return errors if the system backend is unavailable.

## 8. Implementation notes
- system domain should be read-only
- domain logic should not import Kubernetes clients directly
- infrastructure discovery belongs behind ports/adapters
