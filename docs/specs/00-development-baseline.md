# Anarchy Development Baseline

Status: approved-for-implementation baseline draft

## 1. Product baseline
Anarchy is a Go-only, CLI-first, KubeVirt-based lightweight private cloud platform.

Primary design goals:
- common cloud model suitable for AWS/VMware migration semantics
- Kubernetes/KubeVirt runtime substrate
- GitOps-first operational model
- Helm-based packaging
- strict spec-first and TDD-first implementation
- extensibility through ports-and-adapters design

## 2. Mandatory operating assumptions
- deployment packaging is Helm
- deployment lifecycle is GitOps-first
- `kubectl apply` manual mutation is not the normal operational path
- CLI is the first product interface
- future UI must consume the same external API

## 3. Required coding standards
- Go only
- protobuf-first for internal contracts
- HTTP/JSON for external API
- domain/application/ports/adapters/transport separation
- one responsibility per package/module
- no hidden side effects in business logic
- all environment-sensitive behavior must be explicit and configurable

## 4. Required process standards
Before any feature implementation:
1. product/domain spec exists
2. external API spec exists if externally visible
3. proto contract exists if internal boundary exists
4. CLI behavior spec exists if CLI command is affected
5. tests are listed before implementation begins

## 5. Implementation milestones
A. lab and runtime proof
B. codebase skeleton
C. system/node/image domains
D. vm domain MVP
E. diagnose domain
F. storage/network/auth extensibility follow-up
