package subnet

import (
	"encoding/json"
	"net/http"
	"strings"

	appsubnet "github.com/popododo0720/anarchy/internal/application/subnet"
)

type Handler struct {
	service *appsubnet.Service
}

func NewHandler(service *appsubnet.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/subnets", h.ListSubnets)
	mux.HandleFunc("/api/v1/subnets/{name}", h.GetSubnet)
}

func (h *Handler) ListSubnets(w http.ResponseWriter, r *http.Request) {
	subnets, err := h.service.ListSubnets(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "subnet_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, subnets)
}

func (h *Handler) GetSubnet(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	subnet, err := h.service.GetSubnet(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "subnet_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, subnet)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
