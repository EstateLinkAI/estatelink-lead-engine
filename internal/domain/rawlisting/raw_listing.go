package rawlisting

import (
	"encoding/json"
	"time"
)

type RawListing struct {
	ID                 string
	Source             string
	ExternalPropertyID string
	RawPayload         json.RawMessage
	ScrapedAt          *time.Time
	ProcessedAt        *time.Time
	ProcessingStatus   string
	ErrorMessage        *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const (
	StatusPending   = "pending"
	StatusProcessed = "processed"
	StatusFailed    = "failed"
)