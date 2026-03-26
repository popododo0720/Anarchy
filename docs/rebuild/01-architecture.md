# Anarchy Architecture

Status: pre-development baseline

## High-level stack
Layer 0: bare-metal lab host
- current development host: 192.168.0.40
- libvirt/KVM with nested virtualization enabled

Layer 1: guest cluster nodes
- Ubuntu guest VMs on libvirt
- 1 node first, 3 nodes next
- k3s cluster runs inside these VMs

Layer 2: cluster substrate
- k3s
- Kubernetes APIs
- baseline storage/networking

Layer 3: VM runtime
- KubeVirt
- CDI for image/data workflows

Layer 4: Anarchy platform layer
- CLI
- external API
- cluster validation and environment awareness
- domain abstractions for compute/network/storage
- future auth, tenancy, and networking expansion

## Language and service choices
- Go only
- External user-facing API: HTTP/JSON
- Internal contracts: gRPC + protobuf
- No Elixir control-plane in the rebuild

## Architectural style
Anarchy follows a ports-and-adapters style:
- domain: business concepts and invariants
- application: use cases
- ports: interfaces
- adapters: Kubernetes/KubeVirt/storage/auth/network implementations
- transport: HTTP/gRPC boundaries
- CLI: product consumer of the external API

## Why this architecture
- It fits Kubernetes/KubeVirt best
- It keeps the runtime simple
- It allows pluggable backends later
- It keeps semantics stable while implementation evolves
