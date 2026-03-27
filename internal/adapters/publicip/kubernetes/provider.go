package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
)

const (
	attachmentTargetAnnotation = "anarchy.io/attachment-target"
	attachmentVMAnnotation     = "anarchy.io/attachment-vm"
	attachmentNICAnnotation    = "anarchy.io/attachment-nic"
)

type Provider struct {
	runner kexec.Runner
}

func NewProvider(runner kexec.Runner) Provider {
	return Provider{runner: runner}
}

type ovnEIPListResponse struct {
	Items []ovnEIPItem `json:"items"`
}

type ovnFIPListResponse struct {
	Items []ovnFIPItem `json:"items"`
}

type ovnEIPItem struct {
	Metadata struct {
		Name        string            `json:"name"`
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		V4IP string `json:"v4Ip"`
	} `json:"spec"`
	Status struct {
		V4IP string `json:"v4Ip"`
	} `json:"status"`
}

type ovnFIPItem struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		OvnEIP string `json:"ovnEip"`
		IPName string `json:"ipName"`
	} `json:"spec"`
}

type ipListResponse struct {
	Items []ipItem `json:"items"`
}

type ipItem struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		IPAddress   string `json:"ipAddress"`
		V4IPAddress string `json:"v4IpAddress"`
	} `json:"spec"`
}

func (p Provider) ListPublicIPs(ctx context.Context) ([]domainpublicip.PublicIPSummary, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "ovn-eips", "-o", "json")
	if err != nil {
		return nil, err
	}
	var payload ovnEIPListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, err
	}
	fipTargets, err := p.listFIPTargets(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domainpublicip.PublicIPSummary, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, toSummary(item, fipTargets[item.Metadata.Name]))
	}
	return items, nil
}

func (p Provider) GetPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "ovn-eips", name, "-o", "json")
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	var item ovnEIPItem
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	targetIP, hasFIP, err := p.getFIPTarget(ctx, name)
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	return toDetail(item, targetIP, hasFIP), nil
}

func (p Provider) AttachPublicIP(ctx context.Context, req domainpublicip.AttachPublicIPRequest) (domainpublicip.PublicIPDetail, error) {
	resolvedTargetIP, err := p.resolveIPAddress(ctx, req.TargetIPAddress)
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	current, err := p.GetPublicIP(ctx, req.Name)
	if err == nil && current.Realized && current.AttachmentTarget == req.AttachmentTarget && current.TargetIPAddress == resolvedTargetIP {
		return current, nil
	}
	target, err := domainpublicip.ParseAttachmentTarget(req.AttachmentTarget)
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	ipResourceName, err := p.resolveIPResourceName(ctx, resolvedTargetIP)
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	req.TargetIPAddress = ipResourceName
	manifest, err := p.writeFIPManifest(req)
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	defer os.Remove(manifest)
	if _, err := p.runner.Run(ctx, "kubectl", "apply", "-f", manifest); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	patchPayload := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s","%s":"%s","%s":"%s"}}}`,
		attachmentTargetAnnotation, req.AttachmentTarget,
		attachmentVMAnnotation, target.VMName,
		attachmentNICAnnotation, target.NICName,
	)
	if _, err := p.runner.Run(ctx, "kubectl", "patch", "ovn-eips", req.Name, "--type", "merge", "-p", patchPayload); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	return p.GetPublicIP(ctx, req.Name)
}

func (p Provider) DetachPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	current, err := p.GetPublicIP(ctx, name)
	if err == nil && !current.Attached && !current.Realized {
		return current, nil
	}
	if _, err := p.runner.Run(ctx, "kubectl", "delete", "ovn-fips", name); err != nil {
		if !isNotFound(err) && !isResourceUnavailable(err) {
			return domainpublicip.PublicIPDetail{}, err
		}
	}
	patchPayload := fmt.Sprintf(`{"metadata":{"annotations":{"%s":null,"%s":null,"%s":null}}}`,
		attachmentTargetAnnotation,
		attachmentVMAnnotation,
		attachmentNICAnnotation,
	)
	if _, err := p.runner.Run(ctx, "kubectl", "patch", "ovn-eips", name, "--type", "merge", "-p", patchPayload); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	return p.GetPublicIP(ctx, name)
}

func toSummary(item ovnEIPItem, targetIP string) domainpublicip.PublicIPSummary {
	target := item.Metadata.Annotations[attachmentTargetAnnotation]
	realized := targetIP != ""
	return domainpublicip.PublicIPSummary{
		Name:             item.Metadata.Name,
		Address:          publicIPAddress(item),
		Attached:         target != "" || realized,
		Realized:         realized,
		Status:           publicIPStatus(target != "", realized),
		AttachmentTarget: target,
		TargetIPAddress:  targetIP,
	}
}

func toDetail(item ovnEIPItem, targetIP string, hasFIP bool) domainpublicip.PublicIPDetail {
	target := item.Metadata.Annotations[attachmentTargetAnnotation]
	realized := hasFIP
	return domainpublicip.PublicIPDetail{
		Name:             item.Metadata.Name,
		Address:          publicIPAddress(item),
		Attached:         target != "" || realized,
		Realized:         realized,
		Status:           publicIPStatus(target != "", realized),
		AttachmentTarget: target,
		TargetIPAddress:  targetIP,
		Type:             "floating",
	}
}

func publicIPStatus(requested, realized bool) string {
	switch {
	case realized:
		return "realized"
	case requested:
		return "pending"
	default:
		return "detached"
	}
}

func publicIPAddress(item ovnEIPItem) string {
	if item.Status.V4IP != "" {
		return item.Status.V4IP
	}
	return item.Spec.V4IP
}

func (p Provider) listFIPTargets(ctx context.Context) (map[string]string, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "ovn-fips", "-o", "json")
	if err != nil {
		if isNotFound(err) || isResourceUnavailable(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var payload ovnFIPListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, err
	}
	targets := make(map[string]string, len(payload.Items))
	for _, item := range payload.Items {
		key := item.Spec.OvnEIP
		if key == "" {
			key = item.Metadata.Name
		}
		resolved, err := p.resolveIPAddress(ctx, item.Spec.IPName)
		if err != nil {
			continue
		}
		targets[key] = resolved
	}
	return targets, nil
}

func (p Provider) getFIPTarget(ctx context.Context, name string) (string, bool, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "ovn-fips", name, "-o", "json")
	if err != nil {
		if isNotFound(err) || isResourceUnavailable(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var item ovnFIPItem
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return "", false, err
	}
	resolved, err := p.resolveIPAddress(ctx, item.Spec.IPName)
	if err != nil {
		return item.Spec.IPName, true, nil
	}
	return resolved, true, nil
}

func isNotFound(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "notfound") || strings.Contains(message, "not found")
}

func isResourceUnavailable(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "doesn't have a resource type") || strings.Contains(message, "the server could not find the requested resource")
}

func (p Provider) resolveIPResourceName(ctx context.Context, ipAddress string) (string, error) {
	trimmed := strings.TrimSpace(ipAddress)
	if trimmed == "" {
		return "", fmt.Errorf("target ip address is required")
	}
	out, err := p.runner.Run(ctx, "kubectl", "get", "ips.kubeovn.io", "-A", "-o", "json")
	if err != nil {
		return "", err
	}
	var payload ipListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return "", err
	}
	for _, item := range payload.Items {
		candidate := item.Spec.V4IPAddress
		if candidate == "" {
			candidate = item.Spec.IPAddress
		}
		if candidate == trimmed {
			return item.Metadata.Name, nil
		}
	}
	return "", fmt.Errorf("kube-ovn ip resource not found for target ip %q", trimmed)
}

func (p Provider) resolveIPAddress(ctx context.Context, ipRef string) (string, error) {
	trimmed := strings.TrimSpace(ipRef)
	if trimmed == "" {
		return "", fmt.Errorf("target ip address is required")
	}
	out, err := p.runner.Run(ctx, "kubectl", "get", "ips.kubeovn.io", trimmed, "-o", "json")
	if err == nil {
		var item ipItem
		if json.Unmarshal([]byte(out), &item) == nil {
			if item.Spec.V4IPAddress != "" {
				return item.Spec.V4IPAddress, nil
			}
			if item.Spec.IPAddress != "" {
				return item.Spec.IPAddress, nil
			}
		}
	}
	return trimmed, nil
}

func (p Provider) writeFIPManifest(req domainpublicip.AttachPublicIPRequest) (string, error) {
	file, err := os.CreateTemp("", "anarchy-publicip-fip-*.yaml")
	if err != nil {
		return "", err
	}
	manifest := fmt.Sprintf(`apiVersion: kubeovn.io/v1
kind: OvnFip
metadata:
  name: %s
spec:
  ovnEip: %s
  ipType: ip
  ipName: %s
`, req.Name, req.Name, req.TargetIPAddress)
	if _, err := file.WriteString(manifest); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}
