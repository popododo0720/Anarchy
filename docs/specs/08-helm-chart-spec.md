# Anarchy Helm Chart Specification

Status: implementation draft

## 1. Helm purpose
Helm is the packaging and parameterization mechanism for cluster deployment.

## 2. Initial chart responsibility
Initial chart should package:
- Anarchy API server
- required service account/RBAC
- configuration references
- service/deployment objects

Future chart responsibility may include additional internal components if architecture grows.

## 3. Chart rules
- chart values express deployment-time configuration only
- business logic must not live in chart templates
- defaults must be safe and minimal
- versioning must allow GitOps promotion by chart version and values change

## 4. Chart layout
- `charts/anarchy/Chart.yaml`
- `charts/anarchy/values.yaml`
- `charts/anarchy/templates/`
- optional `values-lab.yaml`, `values-dev.yaml` later if needed
