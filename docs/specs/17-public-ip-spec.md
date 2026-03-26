# Anarchy Public IP and Floating IP Specification

Status: implementation draft

## 1. Goal
Public reachability semantics must attach to a specific NIC, not to the VM as an undifferentiated object.

## 2. Initial resources
- PublicIP
- PublicIPAttachment

## 3. Semantics
### PublicIP
Represents an externally reachable address resource.
It may be unattached or attached.

### Floating IP behavior
A floating-style PublicIP should be re-attachable between NICs without changing the VM's private network identity.

### Attachment target
The attachment target must identify a VM NIC, not only a VM.
Example target shape:
- `vm-name:nic0`

## 4. Initial implementation narrowing
The first implementation may be inventory/read-only.
Write/attach semantics can follow after the domain and API are stabilized.

## 5. Rules
- PublicIP must remain independent from VM lifecycle
- PublicIP attachment must reference a NIC role/name
- external NICs are the default attachment candidates in common cases
- internal/backend/storage NICs may be disallowed for public attachment by policy later
