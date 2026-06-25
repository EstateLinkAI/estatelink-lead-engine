package ingestlisting

import (
	"context"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/scorestrategies"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/strategy"
)

type ListingRepository interface {
	Create(ctx context.Context, l listing.Listing) (listing.Listing, error)
}

type LeadScoreRepository interface {
	Create(ctx context.Context, score lead.Score) (lead.Score, error)
}

type Result struct {
	Listing        listing.Listing          `json:"listing"`
	Score          lead.Score               `json:"score"`
	StrategyScores []strategy.StrategyScore `json:"strategyScores,omitempty"`
}

type UseCase struct {
	listingRepo    ListingRepository
	leadScoreRepo LeadScoreRepository
	strategyScorer *scorestrategies.UseCase
}

func NewUseCase(
	listingRepo ListingRepository,
	leadScoreRepo LeadScoreRepository,
	strategyScorer *scorestrategies.UseCase,
) *UseCase {
	return &UseCase{
		listingRepo:    listingRepo,
		leadScoreRepo: leadScoreRepo,
		strategyScorer: strategyScorer,
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

	var strategyScores []strategy.StrategyScore

	if uc.strategyScorer != nil {
		strategyScores, err = uc.strategyScorer.Execute(ctx, scorestrategies.ListingInput{
			ListingID:       savedListing.ID,
			Price:           savedListing.Price,
			RentalEstimate:  savedListing.RentalEstimate,
			Bedrooms:        savedListing.Bedrooms,
			PropertyType:    savedListing.PropertyType,
			City:            savedListing.City,
			PostcodeArea:    savedListing.PostcodeArea,
			DaysOnMarket:    savedListing.DaysOnMarket,
		})
		if err != nil {
			return Result{}, err
		}
	}

	return Result{
		Listing:        savedListing,
		Score:          savedScore,
		StrategyScores: strategyScores,
	}, nil
}