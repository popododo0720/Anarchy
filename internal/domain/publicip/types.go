package publicip

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
