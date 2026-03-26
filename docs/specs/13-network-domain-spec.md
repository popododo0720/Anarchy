# Anarchy Network Domain Specification

Status: implementation draft

## 1. Goal
Anarchy network resources must preserve cloud-style semantics while mapping cleanly onto Kube-OVN primitives.

## 2. Initial resource set
Initial network-domain resources:
- Network
- Subnet
- NetworkInterface

## 3. Semantics
### Network
Represents a top-level logical cloud network domain.
Initial Kube-OVN mapping:
- VPC-backed network domain
- default implementation starts with `ovn-cluster`

### Subnet
Represents an address range within a Network.
Initial Kube-OVN mapping:
- Kube-OVN `Subnet` CRD
- fields surfaced initially:
  - name
  - cidr
  - gateway
  - protocol
  - provider
  - vlan
  - vpc/networkRef
  - namespaces if scoped
  - private flag / default flag semantics later

### NetworkInterface
Represents workload attachment to a subnet.
Initial Kube-OVN/KubeVirt mapping:
- one primary NIC per VM
- maps to KubeVirt network/interface plus the launcher pod's OVN allocation

## 4. Initial implementation narrowing
The first code increment may start with:
- subnet list
- subnet show
- vm create still using one primary network attachment

## 5. Rules
- network and subnet must be first-class resources, not hidden VM fields
- VM requests should evolve toward `networkAttachments` / `subnetRef`
- NIC identity should remain stable even if IP/public-IP semantics expand later
- Kube-OVN implementation details must stay behind adapters
