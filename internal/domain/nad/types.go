package nad

type NADSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
}

type NADDetail struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
	RawConfig string `json:"rawConfig"`
}
