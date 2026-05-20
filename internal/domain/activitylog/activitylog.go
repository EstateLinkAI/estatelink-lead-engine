package activitylog

import "time"

// ActivityLog represents an action performed by a user in the system
type ActivityLog struct {
	ID          int64                  // BIGSERIAL
	ActorUserID string                 // UUID matching users.id
	Action      string                 // e.g., "listing.ingested"
	EntityType  string                 // e.g., "listing", "lead"
	EntityID    int64                  // ID of the entity acted upon
	Metadata    map[string]interface{} // Extra info (JSONB)
	IPAddress   string                 // IP address of actor
	UserAgent   string                 // User agent string
	CreatedAt   time.Time              // Timestamp of the action
}