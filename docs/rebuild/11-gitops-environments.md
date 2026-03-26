# Anarchy GitOps Environments

Status: pre-development baseline

## Goal
Define how Anarchy should be promoted and managed across environments using GitOps.

## Recommended environment layout
- `deploy/gitops/lab/`
- `deploy/gitops/dev/`
- `deploy/gitops/test/`
- later `deploy/gitops/prod/`

## Each environment should declare
- chart version
- image tag/digest
- values overrides
- enabled features
- namespace/release naming

## Promotion rule
Promotion between environments should happen by Git change, not by manual cluster mutation.

## Operator workflow
1. change manifests/values in Git
2. review and merge
3. GitOps reconciler applies desired state
4. CLI/API verifies health

## Product rule
GitOps controls deployment state.
Anarchy controls product behavior.
These should complement each other, not compete.
