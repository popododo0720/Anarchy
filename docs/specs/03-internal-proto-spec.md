# Anarchy Internal Proto Specification

Status: implementation draft

## 1. Proto principles
- internal contracts are protobuf-first
- even if the first server implementation is monolithic, internal contracts must be designed as if they can be split later
- common types should be shared in `anarchy.common.v1`

## 2. Initial proto packages
- `anarchy.common.v1`
- `anarchy.system.v1`
- `anarchy.node.v1`
- `anarchy.image.v1`
- `anarchy.vm.v1`
- `anarchy.network.v1`
- `anarchy.diagnostics.v1`

## 3. Expected services
### SystemService
RPCs:
- GetHealth
- GetVersion
- GetCapabilities

### NodeService
RPCs:
- ListNodes
- GetNode

### ImageService
RPCs:
- ListImages
- GetImage

### VMService
RPCs:
- CreateVM
- ListVMs
- GetVM
- StartVM
- StopVM
- RestartVM
- DeleteVM

### DiagnosticsService
RPCs:
- DiagnoseCluster
- DiagnoseVM

## 4. Common types should cover
- ResourceRef
- Condition
- WorkloadPhase
- ImageRef
- NetworkAttachmentRef
- PrivateIP
- PublicIP
- NodeCapabilitySummary
- ErrorDetail

## 5. Versioning rule
- no breaking change without new version namespace
- proto package names carry explicit version
