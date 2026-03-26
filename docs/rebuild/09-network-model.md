# Anarchy Network Model

Status: pre-development baseline

## Product goal
VMs and Pods should be modeled as workloads within a shared cloud-style networking system.
Users should think in terms of:
- networks
- subnets
- interfaces
- private IPs
- public IPs
- security policies
not in terms of different low-level network stacks per workload type.

## Semantic requirements
Anarchy must support, as part of its core model:
- private IP assignment
- public IP assignment
- floating-IP-style attach/detach semantics
- future dual-IP behavior
- future multi-NIC behavior

## Important rule
These semantics are mandatory in the resource model even if early implementation is simpler.

## Early implementation rule
The first implementation may use a simpler public-IP realization path, as long as the external semantics are preserved.
That means:
- model first
- implementation later

## Workload abstraction
Both VM and Pod workloads should fit the same cloud-network model.
Implementation details may differ through adapters, but the user model should remain consistent.
