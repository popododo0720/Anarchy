package nad

import (
	"encoding/json"
	"net/http"
	"strings"

	appnad "github.com/popododo0720/anarchy/internal/application/nad"
	domainnad "github.com/popododo0720/anarchy/internal/domain/nad"
)

type Handler struct {
	service *appnad.Service
}

func NewHandler(service *appnad.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/nads", h.routeNADs)
	mux.HandleFunc("/api/v1/nads/{namespace}/{name}", h.GetNAD)
}

func (h *Handler) routeNADs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListNADs(w, r)
	case http.MethodPost:
		h.CreateNAD(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) ListNADs(w http.ResponseWriter, r *http.Request) {
	nads, err := h.service.ListNADs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nad_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nads)
}

func (h *Handler) GetNAD(w http.ResponseWriter, r *http.Request) {
	namespace := strings.TrimSpace(r.PathValue("namespace"))
	name := strings.TrimSpace(r.PathValue("name"))
	nad, err := h.service.GetNAD(r.Context(), namespace, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nad_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nad)
}

func (h *Handler) CreateNAD(w http.ResponseWriter, r *http.Request) {
	var req domainnad.CreateNADRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	nad, err := h.service.CreateNAD(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nad_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, nad)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
