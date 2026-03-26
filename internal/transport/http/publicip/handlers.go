package publicip

import (
	"encoding/json"
	"net/http"
	"strings"

	apppublicip "github.com/popododo0720/anarchy/internal/application/publicip"
	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
)

type Handler struct {
	service *apppublicip.Service
}

func NewHandler(service *apppublicip.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/public-ips", h.ListPublicIPs)
	mux.HandleFunc("/api/v1/public-ips/{name}", h.GetPublicIP)
	mux.HandleFunc("/api/v1/public-ips/{name}/attach", h.AttachPublicIP)
	mux.HandleFunc("/api/v1/public-ips/{name}/detach", h.DetachPublicIP)
}

func (h *Handler) ListPublicIPs(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPublicIPs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "public_ip_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetPublicIP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	item, err := h.service.GetPublicIP(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "public_ip_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) AttachPublicIP(w http.ResponseWriter, r *http.Request) {
	var req domainpublicip.AttachPublicIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := h.service.AttachPublicIP(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "public_ip_attach_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, item)
}

func (h *Handler) DetachPublicIP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	item, err := h.service.DetachPublicIP(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "public_ip_detach_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, item)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
