package http

import (
	"encoding/json"
	"net/http"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/logactivity"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/activitylog"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/go-chi/chi/v5"
)

type ListingHandler struct {
	ingestUseCase   *ingestlisting.UseCase
	activityService *logactivity.Service
}

func NewListingHandler(ingestUseCase *ingestlisting.UseCase, activityService *logactivity.Service) *ListingHandler {
	return &ListingHandler{
		ingestUseCase:   ingestUseCase,
		activityService: activityService,
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

	currentUser, ok := GetCurrentUser(r)
	if ok {
		ipAddress, userAgent := requestMetadata(r)
		logActivityBestEffort(r.Context(), h.activityService, activitylog.ActivityLog{
			ActorUserID: currentUser.ID,
			Action:      "listing.ingested",
			EntityType:  "listing",
			EntityID:    result.Listing.ID,
			Metadata: map[string]interface{}{
				"city":            result.Listing.City,
				"postcode_area":   result.Listing.PostcodeArea,
				"property_type":   result.Listing.PropertyType,
				"source_platform": result.Listing.SourcePlatform,
			},
			IPAddress: ipAddress,
			UserAgent: userAgent,
		})
	}

	writeJSON(w, http.StatusCreated, result)
}
