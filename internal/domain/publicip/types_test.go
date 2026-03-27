package publicip_test

import (
	"testing"

	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
)

func TestParseAttachmentTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantVM  string
		wantNIC string
		wantErr bool
	}{
		{name: "valid target", target: "vm1:nic1", wantVM: "vm1", wantNIC: "nic1"},
		{name: "trims whitespace", target: " vm1:nic1 ", wantVM: "vm1", wantNIC: "nic1"},
		{name: "missing separator", target: "vm1", wantErr: true},
		{name: "missing vm name", target: ":nic1", wantErr: true},
		{name: "missing nic name", target: "vm1:", wantErr: true},
		{name: "too many separators", target: "vm1:nic1:extra", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := domainpublicip.ParseAttachmentTarget(tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseAttachmentTarget(%q) error = nil, want error", tt.target)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAttachmentTarget(%q) error = %v", tt.target, err)
			}
			if parsed.VMName != tt.wantVM || parsed.NICName != tt.wantNIC {
				t.Fatalf("ParseAttachmentTarget(%q) = %#v, want vm=%q nic=%q", tt.target, parsed, tt.wantVM, tt.wantNIC)
			}
		})
	}
}
