package publicip

import (
	"fmt"
	"strings"
)

type AttachmentTarget struct {
	VMName  string
	NICName string
}

func ParseAttachmentTarget(raw string) (AttachmentTarget, error) {
	trimmed := strings.TrimSpace(raw)
	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return AttachmentTarget{}, fmt.Errorf("attachment target must use vm:nic format")
	}
	vmName := strings.TrimSpace(parts[0])
	nicName := strings.TrimSpace(parts[1])
	if vmName == "" || nicName == "" {
		return AttachmentTarget{}, fmt.Errorf("attachment target must use vm:nic format")
	}
	return AttachmentTarget{VMName: vmName, NICName: nicName}, nil
}

type AttachPublicIPRequest struct {
	Name             string `json:"name"`
	AttachmentTarget string `json:"attachmentTarget"`
	TargetIPAddress  string `json:"targetIpAddress,omitempty"`
}

type PublicIPSummary struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	Attached         bool   `json:"attached"`
	AttachmentTarget string `json:"attachmentTarget"`
}

type PublicIPDetail struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	Attached         bool   `json:"attached"`
	AttachmentTarget string `json:"attachmentTarget"`
	Type             string `json:"type"`
}
