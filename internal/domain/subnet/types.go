package subnet

type CreateSubnetRequest struct {
	Name       string   `json:"name"`
	CIDR       string   `json:"cidr"`
	Gateway    string   `json:"gateway"`
	Protocol   string   `json:"protocol"`
	Provider   string   `json:"provider"`
	Network    string   `json:"network"`
	Namespaces []string `json:"namespaces,omitempty"`
}

type SubnetSummary struct {
	Name     string `json:"name"`
	CIDR     string `json:"cidr"`
	Gateway  string `json:"gateway"`
	Protocol string `json:"protocol"`
	Network  string `json:"network"`
}

type SubnetDetail struct {
	Name       string   `json:"name"`
	CIDR       string   `json:"cidr"`
	Gateway    string   `json:"gateway"`
	Protocol   string   `json:"protocol"`
	Provider   string   `json:"provider"`
	VLAN       string   `json:"vlan"`
	Network    string   `json:"network"`
	Namespaces []string `json:"namespaces"`
}
