# Anarchy Core Resource Model

Status: pre-development baseline

## Goal
Define a cloud-model-first resource system that supports later migration mapping from AWS/VMware-style environments.

## Core resources
### Workload
A running unit managed by Anarchy.
Types:
- vm
- pod

### Image
A bootable source/template for a workload.

### Volume
A persistent storage unit attachable to workloads.
Backend-specific behavior must be hidden behind adapters.

### Network
A top-level logical network domain.
Comparable in meaning to a VPC-style construct.

### Subnet
An address range within a Network.
Used for private addressing and placement semantics.

### NetworkInterface
A first-class attachment owned by a Workload.
This is critical for migration-friendly modeling.
It is the anchor for:
- private IPs
- security policy attachment
- future public/floating IP attachment

### PrivateIPAssignment
A private address attached to a NetworkInterface.

### PublicIP
An externally reachable address resource.
It must be modeled independently from the workload.

### PublicIPAttachment
A binding between a PublicIP and a NetworkInterface.
This creates floating-IP-style semantics.
Implementation may be NAT-backed or use richer routing later.

### SecurityPolicy
A rule set controlling inbound/outbound communication.
Comparable in meaning to security-group style semantics.

## Semantic rule
The user-visible model must support:
- private/public dual-IP semantics
- floating-IP style attachment semantics
- stable NIC-based modeling
Even if early implementation is simpler than AWS or VMware internals.
