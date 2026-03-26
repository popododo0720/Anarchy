# Anarchy Lab Bootstrap Specification

Status: implementation draft

## Goal
Use the bare-metal host at 192.168.0.40 as the development substrate.

## Current known baseline
- libvirt/KVM available
- nested virtualization enabled
- existing 3-VM lab shape known

## Bootstrap sequence
1. record and preserve current VM inventory
2. destroy current guest VMs if proceeding with rebuild
3. recreate node1 first with same class of VM sizing
4. install k3s server on node1
5. install KubeVirt and CDI
6. validate first nested VM lifecycle
7. recreate node2/node3
8. join them to the cluster
9. validate 3-node behavior

## Exit criteria
- one-node and three-node bootstrap are both reproducible
- steps are scriptable and documentable
- development can rely on a stable lab substrate
