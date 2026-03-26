# Anarchy CLI-First Plan

Status: pre-development baseline

## Why CLI first
- fastest path to a usable product
- easiest way to stabilize API semantics
- simplest path for TDD and repeatable testing
- future web UI can bind to the same API later

## CLI v1 command groups
### system
- `anarchy system health`
- `anarchy system version`
- `anarchy system capabilities`

### node
- `anarchy node list`
- `anarchy node show <name>`

### image
- `anarchy image list`
- `anarchy image show <name>`

### vm
- `anarchy vm create`
- `anarchy vm list`
- `anarchy vm show <name>`
- `anarchy vm start <name>`
- `anarchy vm stop <name>`
- `anarchy vm restart <name>`
- `anarchy vm delete <name>`

### diagnose
- `anarchy diagnose cluster`
- `anarchy diagnose vm <name>`

## v1 completion criteria
- cluster health visible
- nodes inspectable
- images inspectable
- VM lifecycle works end to end
- diagnostics explain failures clearly
