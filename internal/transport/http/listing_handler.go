package http

import (
	"encoding/json"
	"net/http"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/go-chi/chi/v5"
)

type ListingHandler struct {
	ingestUseCase *ingestlisting.UseCase
}

func NewListingHandler(ingestUseCase *ingestlisting.UseCase) *ListingHandler {
	return &ListingHandler{
		ingestUseCase: ingestUseCase,
	}
}

func (h *ListingHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/listings", h.CreateListing)
}

func (h *ListingHandler) CreateListing(w http.ResponseWriter, r *http.Request) {
	var input listing.Listing

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.ingestUseCase.Execute(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create listing",
		})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
