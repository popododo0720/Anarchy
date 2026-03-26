package network

type CreateNetworkRequest struct {
	Name string `json:"name"`
}

type NetworkSummary struct {
	Name          string `json:"name"`
	Default       bool   `json:"default"`
	DefaultSubnet string `json:"defaultSubnet"`
}

type NetworkDetail struct {
	Name          string   `json:"name"`
	Default       bool     `json:"default"`
	Router        string   `json:"router"`
	DefaultSubnet string   `json:"defaultSubnet"`
	Subnets       []string `json:"subnets"`
}
