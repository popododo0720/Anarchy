# Anarchy Development Rules

Status: mandatory before coding

## Mandatory rules
1. Spec first
2. TDD first
3. SOLID and SRP by default
4. Clean-code and explicit boundaries
5. No hidden manual behavior
6. Document before implement

## Feature workflow
For every feature:
1. write/update spec
2. write/update interface contract
3. write failing test
4. confirm failure
5. implement minimum code
6. confirm test passes
7. refactor
8. update docs and verification notes

## Definition of done
A task is only done when:
- spec exists
- tests pass
- docs are updated
- verification steps are recorded
- code respects layering and adapter boundaries

## Extensibility rule
Future backend variation must be expected from day one.
Examples:
- storage: local, NAS/NFS, Ceph, external provider
- auth: bootstrap/local, Keycloak, generic OIDC
- network: simple private network, richer public/floating IP models

Therefore, external-system logic must sit behind ports and adapters.

## Product-interface rule
- CLI is the first official interface
- Future web UI must bind to the same external API
- No separate hidden logic path for the future web UI
