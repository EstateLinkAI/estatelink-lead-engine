package importjob

import "time"

type ImportJob struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"`
	TotalCount     int        `json:"totalCount"`
	ProcessedCount int        `json:"processedCount"`
	FailedCount    int        `json:"failedCount"`
	ErrorMessage    *string    `json:"errorMessage,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)