# Anarchy Domain Model Specification

Status: implementation draft

## 1. Core entities
### Workload
Represents a compute workload under Anarchy.
Initial focus: VM workloads.
Future-compatible with Pod workloads.

### Node
Represents a schedulable infrastructure node with capabilities relevant to Anarchy.

### Image
Represents a bootable template/source.

### Volume
Represents persistent storage requested by a workload.

### Network
Represents a top-level logical cloud network domain.

### Subnet
Represents an address range within a Network.

### NetworkInterface
Represents an attachment between a Workload and a Subnet/Network.
This is the anchor for IP and policy semantics.

### PrivateIPAssignment
Represents a private IP on an interface.

### PublicIP
Represents an externally reachable address resource.

### PublicIPAttachment
Represents a binding between PublicIP and NetworkInterface.
This is how floating-IP style semantics are represented.

### SecurityPolicy
Represents network access control intent.

## 2. Domain rules
- a Workload may have one or more NetworkInterfaces in the long-term model
- a NetworkInterface may have one or more private IPs in the long-term model
- a PublicIP is modeled independently of Workload lifecycle
- a PublicIP may be unattached or attached
- early implementation may simplify realization but not semantics

## 3. Initial implementation narrowing
v1 may initially restrict:
- one workload type in practice: VM
- one primary interface per workload
- one private IP per interface
- public IP semantics reserved in the model but partially implemented later
