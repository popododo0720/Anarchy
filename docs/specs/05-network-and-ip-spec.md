# Anarchy Network and IP Specification

Status: implementation draft

## 1. Product requirement
Anarchy must preserve cloud-style semantics required for migration-friendly design:
- private IP identity
- public IP identity
- future floating-IP style reassignment
- NIC-based network modeling

## 2. Semantics-first rule
The user-visible resource model must support:
- a workload having a private address in a subnet
- a workload optionally having public reachability semantics
- a PublicIP existing independently from a workload
- future attach/detach and reassignment semantics

## 3. Important implementation note
Early implementation does not need to realize every advanced mode immediately.
However, the public/private/floating model must not be excluded from the domain model.

## 4. VM and Pod consistency rule
Users should think in terms of cloud network resources.
They should not need to reason about separate product models for VM versus Pod networking.
Adapter implementations may differ, but resource semantics should be consistent.

## 5. Initial implementation recommendation
- private network semantics first
- public IP/floating semantics modeled immediately
- richer realization phased in later without changing the model
