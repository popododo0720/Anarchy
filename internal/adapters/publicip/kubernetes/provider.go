package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

func (p Provider) ListPublicIPs(ctx context.Context) ([]domainpublicip.PublicIPSummary, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "ovneips.kubeovn.io", "-o", "json")
	if err != nil {
		return nil, err
	}
	var payload ovnEIPListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, err
	}
	items := make([]domainpublicip.PublicIPSummary, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, toSummary(item))
	}
	return items, nil
}

func (p Provider) GetPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "ovneip.kubeovn.io", name, "-o", "json")
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	var item ovnEIPItem
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	return toDetail(item), nil
}

func (p Provider) AttachPublicIP(ctx context.Context, req domainpublicip.AttachPublicIPRequest) (domainpublicip.PublicIPDetail, error) {
	target, err := domainpublicip.ParseAttachmentTarget(req.AttachmentTarget)
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
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
	if _, err := p.runner.Run(ctx, "kubectl", "patch", "ovneip.kubeovn.io", req.Name, "--type", "merge", "-p", patchPayload); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	return p.GetPublicIP(ctx, req.Name)
}

func (p Provider) DetachPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	if _, err := p.runner.Run(ctx, "kubectl", "delete", "ovnfip.kubeovn.io", name); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	patchPayload := fmt.Sprintf(`{"metadata":{"annotations":{"%s":null,"%s":null,"%s":null}}}`,
		attachmentTargetAnnotation,
		attachmentVMAnnotation,
		attachmentNICAnnotation,
	)
	if _, err := p.runner.Run(ctx, "kubectl", "patch", "ovneip.kubeovn.io", name, "--type", "merge", "-p", patchPayload); err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	return p.GetPublicIP(ctx, name)
}

func toSummary(item ovnEIPItem) domainpublicip.PublicIPSummary {
	target := item.Metadata.Annotations[attachmentTargetAnnotation]
	return domainpublicip.PublicIPSummary{
		Name:             item.Metadata.Name,
		Address:          publicIPAddress(item),
		Attached:         target != "",
		AttachmentTarget: target,
	}
}

func toDetail(item ovnEIPItem) domainpublicip.PublicIPDetail {
	target := item.Metadata.Annotations[attachmentTargetAnnotation]
	return domainpublicip.PublicIPDetail{
		Name:             item.Metadata.Name,
		Address:          publicIPAddress(item),
		Attached:         target != "",
		AttachmentTarget: target,
		Type:             "floating",
	}
}

func publicIPAddress(item ovnEIPItem) string {
	if item.Status.V4IP != "" {
		return item.Status.V4IP
	}
	return item.Spec.V4IP
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
