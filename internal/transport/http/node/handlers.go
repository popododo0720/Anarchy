package node

import (
	"encoding/json"
	"net/http"

	appnode "github.com/popododo0720/anarchy/internal/application/node"
)

type Handler struct {
	service *appnode.Service
}

func NewHandler(service *appnode.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/nodes", h.ListNodes)
	mux.HandleFunc("/api/v1/nodes/{name}", h.GetNode)
}

func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.service.ListNodes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "node_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (h *Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	node, err := h.service.GetNode(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "node_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
