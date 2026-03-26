# Kube-OVN Migration Notes

Status: lab precheck complete

## Current lab networking state
The current 3-node k3s lab is using the default k3s/flannel primary CNI.
Observed on node1:
- interface `flannel.1` exists
- interface `cni0` exists
- CNI config file: `/var/lib/rancher/k3s/agent/etc/cni/net.d/10-flannel.conflist`

## Important implication
Kube-OVN as the primary cluster network should not be installed blindly on top of the already-running flannel-based cluster.
That would be a disruptive network migration and can break the currently working 3-node lab.

## Safe interpretation
There are two realistic paths:

### Path A: keep this cluster stable, rebuild networking in a fresh K3s cluster
Recommended for the current lab.

High-level steps:
1. preserve the current validated cluster state
2. recreate the k3s cluster with flannel disabled
3. install Kube-OVN as the primary CNI early in cluster bootstrap
4. reinstall KubeVirt + CDI
5. redeploy Anarchy via Helm
6. revalidate image/vm/diagnose flows

### Path B: install Kube-OVN in non-primary / secondary mode
This may be possible for experimentation but does not satisfy the long-term goal of Kube-OVN as the primary network backend.

## Recommendation
For Anarchy's target architecture, use Kube-OVN as a fresh-cluster primary CNI migration step, not as an in-place live mutation of the currently validated flannel cluster.
