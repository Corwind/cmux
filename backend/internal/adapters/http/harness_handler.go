package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Corwind/cmux/backend/internal/harness"
)

type HarnessHandler struct {
	registry *harness.Registry
}

func NewHarnessHandler(registry *harness.Registry) *HarnessHandler {
	return &HarnessHandler{registry: registry}
}

type harnessResponse struct {
	Type        string `json:"type"`
	SectionName string `json:"section_name"`
	IsDefault   bool   `json:"is_default"`
}

func (h *HarnessHandler) List(w http.ResponseWriter, r *http.Request) {
	types := h.registry.All()

	resp := make([]harnessResponse, 0, len(types))
	for i, t := range types {
		resp = append(resp, harnessResponse{
			Type:        string(t),
			SectionName: h.registry.SectionName(t),
			IsDefault:   i == 0,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}
