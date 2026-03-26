# Anarchy Subnet API and CLI Specification

Status: implementation draft

## 1. CLI
### subnet list
Purpose:
- list Kube-OVN-backed subnets visible to Anarchy

Fields:
- name
- cidr
- gateway
- protocol
- vpc/network

### subnet show <name>
Purpose:
- show detailed subnet state and mapping data

Fields:
- name
- cidr
- gateway
- protocol
- provider
- vlan
- namespaces
- vpc/network

## 2. HTTP API
### GET /api/v1/subnets
Returns subnet summary list.

### GET /api/v1/subnets/{name}
Returns subnet detail.

## 3. Notes
- first implementation is read-only inventory
- create/update/delete semantics can come later
- this allows VM requests to move from `network: default` toward `subnetRef: <name>` without guessing hidden adapter state
