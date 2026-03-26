# Anarchy GitOps Specification

Status: implementation draft

## 1. GitOps-first rule
Git is the source of truth for cluster deployment state.

## 2. Operational model
- Helm defines package structure
- environment-specific state is committed in Git
- a GitOps reconciler applies desired state to the cluster
- CLI/API are used to observe and operate the product, not to replace deployment reconciliation

## 3. Environment structure
Recommended:
- `deploy/gitops/lab/`
- `deploy/gitops/dev/`
- `deploy/gitops/test/`
- later `deploy/gitops/prod/`

## 4. Promotion model
Environment promotion should happen by Git change:
- chart version update
- image tag/digest update
- values update

## 5. Anti-patterns
Do not rely on:
- manual in-cluster drift as standard workflow
- imperative kubectl patching as normal release behavior
- undocumented environment mutation outside Git
