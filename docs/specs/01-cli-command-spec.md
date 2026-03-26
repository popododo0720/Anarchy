# Anarchy CLI Command Specification

Status: implementation draft

## 1. CLI principles
- human-usable by default
- script-friendly output modes later
- stable command naming
- errors must explain cause and next action where possible
- CLI must call external API, not bypass it

## 2. Root structure
- `anarchy system ...`
- `anarchy node ...`
- `anarchy image ...`
- `anarchy vm ...`
- `anarchy diagnose ...`

## 3. Command definitions

### system health
Purpose:
- report API reachability, cluster health summary, KubeVirt readiness summary

Success output minimum:
- API reachable yes/no
- Kubernetes reachable yes/no
- KubeVirt installed yes/no
- CDI installed yes/no
- nodes ready count

### system version
Purpose:
- print CLI version, API version, server version, supported API versions

### system capabilities
Purpose:
- show enabled capabilities
Examples:
- vm lifecycle
- image inventory
- public IP support
- storage backends available

### node list
Purpose:
- list cluster nodes relevant to Anarchy
Fields:
- name
- role/class
- schedulable
- ready
- virtualization capable
- workload capability summary

### node show <name>
Purpose:
- detailed single-node information
Fields include:
- labels/taints summary
- virtualization capability
- storage/network capability summary
- conditions summary

### image list
Purpose:
- list known images/templates
Fields:
- name
- source type
- ready state
- size if known
- tags if any

### image show <name>
Purpose:
- detailed image/template information

### vm create
Purpose:
- create a VM from declared inputs
Required inputs initially:
- name
- image
- cpu
- memory
- network/subnet attachment
Optional initial inputs:
- volume size
- node placement hint
- public IP request flag later

### vm list
Purpose:
- list known VMs
Fields:
- name
- phase
- node
- private IP summary
- public IP summary if any
- image

### vm show <name>
Purpose:
- detailed VM state
Fields include:
- phase
- image
- volumes
- interfaces
- private/public IP data
- recent relevant conditions summary

### vm start|stop|restart <name>
Purpose:
- lifecycle actions with clear state reporting

### vm delete <name>
Purpose:
- delete VM with explicit behavior around attached resources stated in API spec

### diagnose cluster
Purpose:
- explain blockers for overall platform readiness
Sources include:
- API health
- Kubernetes readiness
- KubeVirt readiness
- CDI readiness
- node capability gaps

### diagnose vm <name>
Purpose:
- explain why a VM is not healthy or not progressing
Sources include:
- VM status/conditions
- related DataVolume state if relevant
- node placement issues if relevant
- network or image readiness if relevant

## 4. Output modes
Initial mode:
- human-readable tables/text

Later modes:
- json
- yaml

## 5. Error style
Errors must follow this pattern where possible:
- what failed
- why it failed
- what the user can check next
