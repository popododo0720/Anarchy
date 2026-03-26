# Anarchy Implementation Phases

Status: implementation draft

## Phase 0
- finalize docs/spec baseline
- preserve old repo history
- establish main/dev workflow

## Phase 1
- rebuild 1-node lab
- install k3s
- install KubeVirt + CDI
- prove nested VM lifecycle

## Phase 2
- expand lab to 3 nodes
- validate placement and environment behavior

## Phase 3
- initialize Go codebase skeleton
- add proto generation
- add lint/test scaffolding
- add CLI root/app skeleton
- add API skeleton

## Phase 4
Implement domains in order:
1. system
2. node
3. image
4. vm
5. diagnose

For each domain:
- finalize domain spec
- finalize API spec
- finalize proto contract
- finalize CLI spec
- write tests first
- implement

## Phase 5
- enforce extensibility seams for storage/network/auth/image
- keep one real implementation per seam at first
