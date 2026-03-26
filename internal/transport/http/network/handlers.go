package network

import (
	"encoding/json"
	"net/http"
	"strings"

	appnetwork "github.com/popododo0720/anarchy/internal/application/network"
)

type Handler struct {
	service *appnetwork.Service
}

func NewHandler(service *appnetwork.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/networks", h.ListNetworks)
	mux.HandleFunc("/api/v1/networks/{name}", h.GetNetwork)
}

func (h *Handler) ListNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := h.service.ListNetworks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "network_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, networks)
}

func (h *Handler) GetNetwork(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	network, err := h.service.GetNetwork(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "network_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, network)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
