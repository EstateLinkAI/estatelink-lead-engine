package ingestlisting

import (
	"context"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
)

type ListingRepository interface {
	Create(ctx context.Context, l listing.Listing) (listing.Listing, error)
}

type LeadScoreRepository interface {
	Create(ctx context.Context, score lead.Score) (lead.Score, error)
}

type Result struct {
	Listing listing.Listing `json:"listing"`
	Score   lead.Score      `json:"score"`
}

type UseCase struct {
	listingRepo   ListingRepository
	leadScoreRepo LeadScoreRepository
}

func NewUseCase(listingRepo ListingRepository, leadScoreRepo LeadScoreRepository) *UseCase {
	return &UseCase{
		listingRepo:   listingRepo,
		leadScoreRepo: leadScoreRepo,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input listing.Listing) (Result, error) {
	normalisedListing := listing.Normalise(input)

	savedListing, err := uc.listingRepo.Create(ctx, normalisedListing)
	if err != nil {
		return Result{}, err
	}

	score := lead.CalculateScore(savedListing)

	savedScore, err := uc.leadScoreRepo.Create(ctx, score)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Listing: savedListing,
		Score:   savedScore,
	}, nil
}
