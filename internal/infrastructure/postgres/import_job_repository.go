package postgres

import (
	"context"
	"fmt"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/importjob"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImportJobRepository struct {
	db *pgxpool.Pool
}

func NewImportJobRepository(db *pgxpool.Pool) *ImportJobRepository {
	return &ImportJobRepository{db: db}
}

func (r *ImportJobRepository) Create(ctx context.Context, totalCount int) (importjob.ImportJob, error) {
	const query = `
		INSERT INTO import_jobs (
			status,
			total_count
		)
		VALUES ($1, $2)
		RETURNING
			id::text,
			status,
			total_count,
			processed_count,
			failed_count,
			error_message,
			created_at,
			started_at,
			completed_at,
			updated_at;
	`

	var job importjob.ImportJob

	err := r.db.QueryRow(
		ctx,
		query,
		importjob.StatusQueued,
		totalCount,
	).Scan(
		&job.ID,
		&job.Status,
		&job.TotalCount,
		&job.ProcessedCount,
		&job.FailedCount,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		return importjob.ImportJob{}, fmt.Errorf("create import job: %w", err)
	}

	return job, nil
}

func (r *ImportJobRepository) List(ctx context.Context, limit int) ([]importjob.ImportJob, error) {
	const query = `
		SELECT
			id::text,
			status,
			total_count,
			processed_count,
			failed_count,
			error_message,
			created_at,
			started_at,
			completed_at,
			updated_at
		FROM import_jobs
		ORDER BY created_at DESC
		LIMIT $1;
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list import jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]importjob.ImportJob, 0, limit)

	for rows.Next() {
		var job importjob.ImportJob

		if err := rows.Scan(
			&job.ID,
			&job.Status,
			&job.TotalCount,
			&job.ProcessedCount,
			&job.FailedCount,
			&job.ErrorMessage,
			&job.CreatedAt,
			&job.StartedAt,
			&job.CompletedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan import job: %w", err)
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list import jobs: %w", err)
	}

	return jobs, nil
}

func (r *ImportJobRepository) GetByID(ctx context.Context, id string) (importjob.ImportJob, error) {
	const query = `
		SELECT
			id::text,
			status,
			total_count,
			processed_count,
			failed_count,
			error_message,
			created_at,
			started_at,
			completed_at,
			updated_at
		FROM import_jobs
		WHERE id = $1;
	`

	var job importjob.ImportJob

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.Status,
		&job.TotalCount,
		&job.ProcessedCount,
		&job.FailedCount,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return importjob.ImportJob{}, fmt.Errorf("import job not found")
		}

		return importjob.ImportJob{}, fmt.Errorf("get import job by id: %w", err)
	}

	return job, nil
}

func (r *ImportJobRepository) MarkProcessing(ctx context.Context, id string) error {
	const query = `
		UPDATE import_jobs
		SET
			status = $2,
			started_at = COALESCE(started_at, NOW()),
			updated_at = NOW()
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id, importjob.StatusProcessing)
	if err != nil {
		return fmt.Errorf("mark import job processing: %w", err)
	}

	return nil
}

func (r *ImportJobRepository) IncrementProcessed(ctx context.Context, id string) error {
	const query = `
		UPDATE import_jobs
		SET
			processed_count = processed_count + 1,
			updated_at = NOW()
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("increment import job processed count: %w", err)
	}

	return nil
}

func (r *ImportJobRepository) IncrementFailed(ctx context.Context, id string) error {
	const query = `
		UPDATE import_jobs
		SET
			failed_count = failed_count + 1,
			updated_at = NOW()
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("increment import job failed count: %w", err)
	}

	return nil
}

func (r *ImportJobRepository) MarkCompleted(ctx context.Context, id string) error {
	const query = `
		UPDATE import_jobs
		SET
			status = $2,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id, importjob.StatusCompleted)
	if err != nil {
		return fmt.Errorf("mark import job completed: %w", err)
	}

	return nil
}

func (r *ImportJobRepository) MarkFailed(ctx context.Context, id string, reason string) error {
	const query = `
		UPDATE import_jobs
		SET
			status = $2,
			error_message = $3,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id, importjob.StatusFailed, reason)
	if err != nil {
		return fmt.Errorf("mark import job failed: %w", err)
	}

	return nil
}

// MarkCancelled marks the job cancelled only if it hasn't already reached a
// terminal state, so a slow cancel request can't override a result that
// finished in the meantime.
func (r *ImportJobRepository) MarkCancelled(ctx context.Context, id string) error {
	const query = `
		UPDATE import_jobs
		SET
			status = $2,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
			AND status NOT IN ($3, $4, $5);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		id,
		importjob.StatusCancelled,
		importjob.StatusCompleted,
		importjob.StatusFailed,
		importjob.StatusCancelled,
	)
	if err != nil {
		return fmt.Errorf("mark import job cancelled: %w", err)
	}

	return nil
}