package nad

type CreateNADRequest struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Type         string `json:"type"`
	Provider     string `json:"provider"`
	ServerSocket string `json:"serverSocket"`
}

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
