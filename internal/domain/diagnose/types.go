package diagnose

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ClusterReport struct {
	Status   string   `json:"status"`
	Findings []string `json:"findings"`
	Checks   []Check  `json:"checks,omitempty"`
}

type VMReport struct {
	Name     string   `json:"name"`
	Phase    string   `json:"phase"`
	Findings []string `json:"findings"`
	Checks   []Check  `json:"checks,omitempty"`
}

type PublicIPReport struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Reason   string   `json:"reason,omitempty"`
	Code     string   `json:"code,omitempty"`
	Findings []string `json:"findings"`
	Checks   []Check  `json:"checks,omitempty"`
}
