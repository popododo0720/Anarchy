package vm

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	appvm "github.com/popododo0720/anarchy/internal/application/vm"
	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
)

type Handler struct{ service *appvm.Service }

func NewHandler(service *appvm.Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/vms", h.routeVMs)
	mux.HandleFunc("/api/v1/vms/{name}", h.GetVM)
	mux.HandleFunc("/api/v1/vms/{name}/start", h.StartVM)
	mux.HandleFunc("/api/v1/vms/{name}/stop", h.StopVM)
	mux.HandleFunc("/api/v1/vms/{name}/restart", h.RestartVM)
	mux.HandleFunc("/api/v1/vms/{name}/delete", h.DeleteVM)
}

func (h *Handler) routeVMs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListVMs(w, r)
	case http.MethodPost:
		h.CreateVM(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) CreateVM(w http.ResponseWriter, r *http.Request) {
	var req domainvm.CreateVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	vm, err := h.service.CreateVM(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vm_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, vm)
}
func (h *Handler) ListVMs(w http.ResponseWriter, r *http.Request) {
	vms, err := h.service.ListVMs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vm_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vms)
}
func (h *Handler) GetVM(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	vm, err := h.service.GetVM(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vm_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vm)
}
func (h *Handler) StartVM(w http.ResponseWriter, r *http.Request) { h.action(w, r, h.service.StartVM) }
func (h *Handler) StopVM(w http.ResponseWriter, r *http.Request)  { h.action(w, r, h.service.StopVM) }
func (h *Handler) RestartVM(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.service.RestartVM)
}
func (h *Handler) DeleteVM(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.service.DeleteVM)
}
func (h *Handler) action(w http.ResponseWriter, r *http.Request, fn func(context.Context, string) error) {
	name := strings.TrimSpace(r.PathValue("name"))
	if err := fn(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, "vm_action_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
