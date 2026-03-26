# Anarchy System CLI Specification

Status: implementation-ready draft

## `anarchy system health`
Purpose:
- show quick platform readiness

Minimum output:
- overall status
- Kubernetes reachability
- KubeVirt readiness
- CDI readiness
- ready node count
- warnings if any

Exit-code guidance:
- 0 when healthy or degraded-but-readable
- non-zero when API request itself fails

## `anarchy system version`
Purpose:
- show client/server/platform version context

Minimum output:
- CLI version
- API version
- server version
- Kubernetes version if available
- KubeVirt version if available

## `anarchy system capabilities`
Purpose:
- show what the platform says it supports now

Minimum output:
- vm lifecycle support yes/no
- image inventory support yes/no
- diagnostics support yes/no
- public IP support yes/no
- any additional capability labels

## Formatting rule
Initial output is human-readable.
Future `-o json` may be added later, but not required for first implementation.
