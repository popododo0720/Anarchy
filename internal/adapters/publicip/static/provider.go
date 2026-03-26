package static

import (
	"context"
	"fmt"

	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
)

type Provider struct{}

func NewProvider() Provider { return Provider{} }

func (Provider) ListPublicIPs(context.Context) ([]domainpublicip.PublicIPSummary, error) {
	return []domainpublicip.PublicIPSummary{{Name: "fip-sample", Address: "203.0.113.10", Attached: false, AttachmentTarget: ""}}, nil
}

func (Provider) GetPublicIP(_ context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	if name != "fip-sample" {
		return domainpublicip.PublicIPDetail{}, fmt.Errorf("public ip not found: %s", name)
	}
	return domainpublicip.PublicIPDetail{Name: "fip-sample", Address: "203.0.113.10", Attached: false, AttachmentTarget: "", Type: "floating"}, nil
}
