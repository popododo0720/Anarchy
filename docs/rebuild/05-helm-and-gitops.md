# Anarchy Helm and GitOps Strategy

Status: mandatory deployment baseline

## Deployment model
Anarchy will be packaged and deployed with Helm.
Operational lifecycle should be GitOps-friendly.

## Helm role
Helm charts are responsible for packaging and parameterizing:
- API service(s)
- future control-plane components
- configmaps/secrets references
- RBAC
- service accounts
- optional supporting components if needed later

Helm should not hide business semantics.
Runtime behavior must remain driven by specs and APIs, not chart-only logic.

## GitOps role
GitOps should be the preferred operational mode for cluster-side deployment.
Recommended direction:
- desired cluster state lives in Git
- Helm chart values are committed declaratively
- reconciliation is handled by a GitOps controller later (e.g. Argo CD or Flux)

## GitOps rules
- chart defaults must be safe and minimal
- environment-specific overrides live separately from chart logic
- no manual kubectl patching as a normal workflow
- cluster configuration should be reproducible from Git

## Chart layout recommendation
- `charts/anarchy/Chart.yaml`
- `charts/anarchy/values.yaml`
- `charts/anarchy/templates/...`
- optional env overlays or GitOps app manifests under `deploy/gitops/`

## Environment strategy
Recommended structure later:
- `deploy/gitops/lab/`
- `deploy/gitops/dev/`
- `deploy/gitops/test/`

Each environment should pin:
- chart version
- image tags
- environment values
- feature toggles

## Rule of responsibility
- Helm packages and parameterizes
- GitOps declares and reconciles
- Anarchy API/CLI expresses product behavior
- Docs define how environments are promoted
