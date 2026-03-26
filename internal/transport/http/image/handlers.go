package image

import (
	"encoding/json"
	"net/http"

	appimage "github.com/popododo0720/anarchy/internal/application/image"
)

type Handler struct {
	service *appimage.Service
}

func NewHandler(service *appimage.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/images", h.ListImages)
	mux.HandleFunc("/api/v1/images/{name}", h.GetImage)
}

func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.service.ListImages(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "image_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, images)
}

func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	image, err := h.service.GetImage(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "image_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, image)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message})
}
