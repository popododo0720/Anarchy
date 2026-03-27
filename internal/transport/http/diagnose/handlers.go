package diagnose

import (
	"encoding/json"
	"net/http"
	"strings"

	appdiag "github.com/popododo0720/anarchy/internal/application/diagnose"
)

type Handler struct{ service *appdiag.Service }

func NewHandler(service *appdiag.Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/diagnose/cluster", h.DiagnoseCluster)
	mux.HandleFunc("/api/v1/diagnose/public-ips/{name}", h.DiagnosePublicIP)
	mux.HandleFunc("/api/v1/diagnose/vms/{name}", h.DiagnoseVM)
}

func (h *Handler) DiagnoseCluster(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.DiagnoseCluster(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "diagnose_cluster_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) DiagnoseVM(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	report, err := h.service.DiagnoseVM(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "diagnose_vm_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) DiagnosePublicIP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	report, err := h.service.DiagnosePublicIP(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "diagnose_public_ip_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
