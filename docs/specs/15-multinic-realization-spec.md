# Anarchy Multi-NIC Realization Specification

Status: implementation draft

## 1. Problem statement
Anarchy's domain model now supports `networkAttachments`, but the current runtime implementation only works for a single pod-style network attachment.
A naive attempt to attach two pod networks with KubeVirt masquerade interfaces fails.

## 2. Key runtime rule
For KubeVirt on Kubernetes:
- one primary pod network may use pod/masquerade style networking
- additional NICs require secondary network realization
- the expected realization path is Multus + NetworkAttachmentDefinition (NAD)

## 3. Mapping model
### Primary attachment
Anarchy model:
- `primary=true`
- network/subnet semantics backed by the cluster's main Kube-OVN pod network

Runtime mapping:
- KubeVirt `pod` network
- KubeVirt masquerade interface
- launcher pod gets OVN IP allocation on the primary subnet

### Secondary attachment
Anarchy model:
- `primary=false`
- additional NIC attached to another subnet/network

Runtime mapping target:
- a Multus `NetworkAttachmentDefinition`
- KubeVirt `multus` network reference
- KubeVirt bridge/binding choice determined by implementation policy

## 4. Required implementation phases
### Phase A
- keep one primary attachment working
- allow multiple attachments in the API model
- expose them in VM outputs

### Phase B
- add NAD inventory and mapping rules
- distinguish `primary` vs `secondary` attachment realization
- generate KubeVirt network spec using `pod` for primary and `multus` for secondary

### Phase C
- add attachment-level IP/state reporting
- support explicit secondary subnet selection and validation

## 5. Important validation rule
A subnet alone is not enough for a secondary NIC.
A secondary attachment must resolve to a valid NAD or equivalent secondary network object that KubeVirt can consume through Multus.

## 6. Near-term recommendation
Next code increment should introduce:
- NAD inventory domain or adapter
- attachment realization policy in the VM adapter
- validation errors when a request asks for multiple attachments but no valid secondary realization path exists
