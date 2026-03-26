# Anarchy Testing Strategy

Status: mandatory before coding

## Principles
- tests are written before implementation whenever practical
- tests follow the feature spec and contract
- each layer is tested at the right level

## Test layers
### Unit tests
Validate:
- domain rules
- validation logic
- command parsing/output logic
- adapter-local behavior

### Contract tests
Validate:
- protobuf contracts
- external API request/response behavior
- error semantics

### Integration tests
Validate:
- API to adapter flow
- Kubernetes/KubeVirt integration paths
- CLI to API behavior

### Environment/e2e tests
Validate:
- 1-node bootstrap
- 3-node bootstrap
- VM create/delete/recreate flow

## Rules
- every feature must name its test targets in the spec
- no feature is complete without at least one automated test path
- infrastructure steps that cannot be fully automated yet must still have repeatable verification docs
