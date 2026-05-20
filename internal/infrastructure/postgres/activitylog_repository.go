package postgres

import (
	"context"
	"encoding/json"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/activitylog"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivityLogRepository struct {
	DB *pgxpool.Pool
}

func NewActivityLogRepository(db *pgxpool.Pool) *ActivityLogRepository {
	return &ActivityLogRepository{
		DB: db,
	}
}

// Insert adds a new activity log to the database.
func (r *ActivityLogRepository) Insert(ctx context.Context, log activitylog.ActivityLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO activity_log
		(actor_user_id, action, entity_type, entity_id, metadata, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.DB.Exec(ctx, query,
		log.ActorUserID,
		log.Action,
		log.EntityType,
		log.EntityID,
		metadataJSON,
		log.IPAddress,
		log.UserAgent,
	)

	return err
}

// List retrieves activity logs with limit and offset.
func (r *ActivityLogRepository) List(ctx context.Context, limit, offset int) ([]activitylog.ActivityLog, error) {
	query := `
		SELECT id, actor_user_id, action, entity_type, entity_id, metadata, ip_address, user_agent, created_at
		FROM activity_log
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.DB.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]activitylog.ActivityLog, 0)

	for rows.Next() {
		var log activitylog.ActivityLog
		var metadataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.ActorUserID,
			&log.Action,
			&log.EntityType,
			&log.EntityID,
			&metadataJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
				return nil, err
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *ActivityLogRepository) GetByID(ctx context.Context, id int64) (activitylog.ActivityLog, error) {
	var log activitylog.ActivityLog
	var metadataJSON []byte

	query := `
		SELECT id, actor_user_id, action, entity_type, entity_id, metadata, ip_address, user_agent, created_at
		FROM activity_log
		WHERE id = $1
	`

	err := r.DB.QueryRow(ctx, query, id).Scan(
		&log.ID,
		&log.ActorUserID,
		&log.Action,
		&log.EntityType,
		&log.EntityID,
		&metadataJSON,
		&log.IPAddress,
		&log.UserAgent,
		&log.CreatedAt,
	)
	if err != nil {
		return activitylog.ActivityLog{}, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
			return activitylog.ActivityLog{}, err
		}
	}

	return log, nil
}