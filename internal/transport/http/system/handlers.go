package system

import (
	"encoding/json"
	"net/http"

	appsystem "github.com/popododo0720/anarchy/internal/application/system"
)

type Handler struct {
	service *appsystem.Service
}

func NewHandler(service *appsystem.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/system/health", h.Health)
	mux.HandleFunc("/api/v1/system/version", h.Version)
	mux.HandleFunc("/api/v1/system/capabilities", h.Capabilities)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetHealth(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "system_health_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":              summary.Status,
		"apiReachable":        summary.APIReachable,
		"kubernetesReachable": summary.KubernetesReachable,
		"kubevirtInstalled":   summary.KubeVirtInstalled,
		"kubevirtReady":       summary.KubeVirtReady,
		"cdiInstalled":        summary.CDIInstalled,
		"cdiReady":            summary.CDIReady,
		"totalNodes":          summary.TotalNodes,
		"readyNodes":          summary.ReadyNodes,
		"warnings":            summary.Warnings,
	})
}

func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetVersion(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "system_version_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cliVersion":           summary.CLIVersion,
		"apiVersion":           summary.APIVersion,
		"serverVersion":        summary.ServerVersion,
		"supportedApiVersions": summary.SupportedAPIVersions,
		"kubernetesVersion":    summary.KubernetesVersion,
		"kubevirtVersion":      summary.KubeVirtVersion,
	})
}

func (h *Handler) Capabilities(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetCapabilities(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "system_capabilities_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"vmLifecycleSupported":    summary.VMLifecycleSupported,
		"imageInventorySupported": summary.ImageInventorySupported,
		"diagnosticsSupported":    summary.DiagnosticsSupported,
		"publicIpSupported":       summary.PublicIPSupported,
		"capabilities":            summary.Capabilities,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
	})
}
