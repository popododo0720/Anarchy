package network

import (
	"encoding/json"
	"net/http"
	"strings"

	appnetwork "github.com/popododo0720/anarchy/internal/application/network"
	domainnetwork "github.com/popododo0720/anarchy/internal/domain/network"
)

type Handler struct {
	service *appnetwork.Service
}

func NewHandler(service *appnetwork.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/networks", h.routeNetworks)
	mux.HandleFunc("/api/v1/networks/{name}", h.GetNetwork)
}

func (h *Handler) routeNetworks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListNetworks(w, r)
	case http.MethodPost:
		h.CreateNetwork(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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

func (h *Handler) CreateNetwork(w http.ResponseWriter, r *http.Request) {
	var req domainnetwork.CreateNetworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	network, err := h.service.CreateNetwork(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "network_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, network)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
