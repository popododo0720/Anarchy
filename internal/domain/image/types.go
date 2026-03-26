package image

type ImageSummary struct {
	Name       string `json:"name"`
	SourceType string `json:"sourceType"`
	Ready      bool   `json:"ready"`
	Size       string `json:"size"`
}

type ImageDetail struct {
	Name        string   `json:"name"`
	SourceType  string   `json:"sourceType"`
	Ready       bool     `json:"ready"`
	Size        string   `json:"size"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}
