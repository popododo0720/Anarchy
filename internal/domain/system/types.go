package system

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

type HealthSummary struct {
	Status              HealthStatus
	APIReachable        bool
	KubernetesReachable bool
	KubeVirtInstalled   bool
	KubeVirtReady       bool
	CDIInstalled        bool
	CDIReady            bool
	TotalNodes          int
	ReadyNodes          int
	Warnings            []string
}

type VersionSummary struct {
	CLIVersion           string
	APIVersion           string
	ServerVersion        string
	SupportedAPIVersions []string
	KubernetesVersion    string
	KubeVirtVersion      string
}

type CapabilitiesSummary struct {
	VMLifecycleSupported    bool
	ImageInventorySupported bool
	DiagnosticsSupported    bool
	PublicIPSupported       bool
	Capabilities            []string
}
