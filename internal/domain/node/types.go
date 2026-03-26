package node

type NodeSummary struct {
	Name                  string `json:"name"`
	Class                 string `json:"class"`
	Ready                 bool   `json:"ready"`
	Schedulable           bool   `json:"schedulable"`
	VirtualizationCapable bool   `json:"virtualizationCapable"`
}

type NodeDetail struct {
	Name                  string   `json:"name"`
	Class                 string   `json:"class"`
	Ready                 bool     `json:"ready"`
	Schedulable           bool     `json:"schedulable"`
	VirtualizationCapable bool     `json:"virtualizationCapable"`
	Capabilities          []string `json:"capabilities"`
}
