# Anarchy System Testing Specification

Status: implementation-ready draft

## Unit tests
Must cover:
- health summary mapping to CLI output
- version summary mapping to CLI output
- capability summary mapping to CLI output
- API handler response serialization
- error mapping behavior

## Contract tests
Must cover:
- gRPC response fields for health/version/capabilities
- HTTP response field shapes for health/version/capabilities

## Integration tests
Must cover at minimum:
- API server wired to a fake system backend
- CLI calling API and printing expected output

## Environment-aware tests later
When lab is ready, add:
- health behavior against real Kubernetes/KubeVirt environment

## TDD order
1. CLI output tests
2. API handler tests
3. application service tests
4. adapter tests
5. integration wiring tests
