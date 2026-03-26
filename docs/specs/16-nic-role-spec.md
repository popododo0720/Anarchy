# Anarchy NIC Role Specification

Status: implementation draft

## 1. Goal
Anarchy NICs must express user-visible traffic intent, not just attachment existence.
NIC role is the semantic hint that explains what a NIC is for.

## 2. Initial role set
### external
Purpose:
- north-south traffic
- public or edge-facing connectivity
- default egress path in the common case

Expected early realization:
- primary OVN pod network
- or provider-backed external network later

### internal
Purpose:
- east-west traffic between workloads
- private application traffic
- tenant-internal connectivity

Expected early realization:
- secondary subnet/NAD-backed network

### backend
Purpose:
- private service backend communication
- app-to-db or internal API traffic

Expected early realization:
- secondary private network attachment
- often no direct public reachability

### storage
Purpose:
- storage replication, backup, block/file service traffic
- isolation from application and public traffic

Expected early realization:
- dedicated secondary network attachment
- may later map to provider/underlay or storage-specific VLANs

### provider
Purpose:
- direct provider-network / underlay attachment
- physical/VLAN/bridge-oriented connectivity

Expected later realization:
- provider network or physical bridge style mapping
- may require host-specific capability and scheduling checks

## 3. Rules
- every NIC may optionally declare a role
- if omitted, implementation may infer `external` for primary and `internal` for secondary
- public/floating/vip attachments should target a specific NIC, not the VM as a whole
- role does not replace network/subnet identity; it complements it

## 4. Recommended VM patterns
### Simple VM
- nic0 role=external

### Split traffic VM
- nic0 role=external
- nic1 role=internal

### Service VM
- nic0 role=external
- nic1 role=backend

### Storage-heavy VM
- nic0 role=external
- nic1 role=storage

## 5. Follow-up implementation work
- expose role consistently in list/show outputs
- support policy and routing semantics per role later
- enforce provider-role validation when provider-network realization is introduced
