package vm

type NetworkAttachment struct {
	Name      string `json:"name"`
	Network   string `json:"network"`
	SubnetRef string `json:"subnetRef,omitempty"`
	Primary   bool   `json:"primary"`
}

type CreateVMRequest struct {
	Name               string              `json:"name"`
	Image              string              `json:"image"`
	CPU                int                 `json:"cpu"`
	Memory             string              `json:"memory"`
	Network            string              `json:"network"`
	SubnetRef          string              `json:"subnetRef,omitempty"`
	NetworkAttachments []NetworkAttachment `json:"networkAttachments,omitempty"`
}

type VMSummary struct {
	Name      string `json:"name"`
	Phase     string `json:"phase"`
	Image     string `json:"image"`
	PrivateIP string `json:"privateIp"`
}

type VMDetail struct {
	Name      string `json:"name"`
	Phase     string `json:"phase"`
	Image     string `json:"image"`
	CPU       int    `json:"cpu"`
	Memory    string `json:"memory"`
	Network   string `json:"network"`
	PrivateIP string `json:"privateIp"`
}
