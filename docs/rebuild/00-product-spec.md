# Anarchy Product Specification

Status: pre-development baseline

## Purpose
Anarchy is a lightweight private cloud platform built on Kubernetes and KubeVirt.

It is designed to:
- run first in a small lab environment
- start with a CLI-first product surface
- expose a common cloud model suitable for future migration from AWS- and VMware-like environments
- remain extensible through clean interfaces and adapter-based design

## Core product direction
- Linux-first
- Go-only codebase
- Kubernetes-native substrate
- KubeVirt as VM runtime layer
- CLI first, web later
- spec-first and TDD-first development
- Helm-delivered application packaging
- GitOps-friendly operational model

## Non-goals for early versions
- Full AWS API compatibility from day one
- Full VMware API compatibility from day one
- Web UI in v1
- Reimplementation of a custom VM runtime
- Complex enterprise DR/billing in the first milestone

## Product semantics
Anarchy must model cloud resources in a migration-friendly way:
- Workload
- Image
- Volume
- Network
- Subnet
- NetworkInterface
- Private IP
- Public IP
- Floating-IP style attachment semantics
- Security policy
- Placement/capability

The implementation may evolve, but these semantics must remain stable.

## Delivery sequence
1. Documentation/specs
2. 1-node lab bootstrap
3. 3-node lab bootstrap
4. Go codebase skeleton
5. CLI-first API domains
6. Adapter-based backend growth
