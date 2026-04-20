package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"example.com/taskservice/internal/domain/dateonly"
	templatedomain "example.com/taskservice/internal/domain/tasktemplate"
)

type TemplateRepository struct {
	db dbTX
}

func NewTemplateRepository(pool dbTX) *TemplateRepository {
	return &TemplateRepository{db: pool}
}

func newTemplateRepository(db dbTX) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(ctx context.Context, template *templatedomain.Template) (*templatedomain.Template, error) {
	const query = `
		INSERT INTO recurring_task_templates (
			title, description, timezone, start_date, end_date, recurrence_type, recurrence_settings, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, title, description, timezone, start_date, end_date, recurrence_type, recurrence_settings, is_active, created_at, updated_at
	`

	row := r.db.QueryRow(
		ctx,
		query,
		template.Title,
		template.Description,
		template.Timezone,
		template.StartDate.ToTime(time.UTC),
		nullDate(template.EndDate),
		template.RecurrenceType,
		mustMarshalSettings(template.RecurrenceSetting),
		template.IsActive,
		template.CreatedAt,
		template.UpdatedAt,
	)

	return scanTemplate(row)
}

func (r *TemplateRepository) GetByID(ctx context.Context, id int64) (*templatedomain.Template, error) {
	const query = `
		SELECT id, title, description, timezone, start_date, end_date, recurrence_type, recurrence_settings, is_active, created_at, updated_at
		FROM recurring_task_templates
		WHERE id = $1 AND is_active = TRUE
	`

	row := r.db.QueryRow(ctx, query, id)
	template, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, templatedomain.ErrNotFound
		}
		return nil, err
	}

	return template, nil
}

func (r *TemplateRepository) Update(ctx context.Context, template *templatedomain.Template) (*templatedomain.Template, error) {
	const query = `
		UPDATE recurring_task_templates
		SET title = $1,
			description = $2,
			timezone = $3,
			start_date = $4,
			end_date = $5,
			recurrence_type = $6,
			recurrence_settings = $7,
			updated_at = $8
		WHERE id = $9 AND is_active = TRUE
		RETURNING id, title, description, timezone, start_date, end_date, recurrence_type, recurrence_settings, is_active, created_at, updated_at
	`

	row := r.db.QueryRow(
		ctx,
		query,
		template.Title,
		template.Description,
		template.Timezone,
		template.StartDate.ToTime(time.UTC),
		nullDate(template.EndDate),
		template.RecurrenceType,
		mustMarshalSettings(template.RecurrenceSetting),
		template.UpdatedAt,
		template.ID,
	)

	updated, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, templatedomain.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

func (r *TemplateRepository) Deactivate(ctx context.Context, id int64, updatedAt time.Time) error {
	const query = `
		UPDATE recurring_task_templates
		SET is_active = FALSE, updated_at = $2
		WHERE id = $1 AND is_active = TRUE
	`

	result, err := r.db.Exec(ctx, query, id, updatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return templatedomain.ErrNotFound
	}
	return nil
}

func (r *TemplateRepository) List(ctx context.Context) ([]templatedomain.Template, error) {
	return r.listByActivity(ctx, true)
}

func (r *TemplateRepository) ListActive(ctx context.Context) ([]templatedomain.Template, error) {
	return r.listByActivity(ctx, true)
}

func (r *TemplateRepository) listByActivity(ctx context.Context, active bool) ([]templatedomain.Template, error) {
	const query = `
		SELECT id, title, description, timezone, start_date, end_date, recurrence_type, recurrence_settings, is_active, created_at, updated_at
		FROM recurring_task_templates
		WHERE is_active = $1
		ORDER BY id DESC
	`

	rows, err := r.db.Query(ctx, query, active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]templatedomain.Template, 0)
	for rows.Next() {
		template, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}

	return templates, rows.Err()
}

type templateScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(scanner templateScanner) (*templatedomain.Template, error) {
	var (
		template           templatedomain.Template
		startDate          time.Time
		endDate            pgtype.Date
		recurrenceType     string
		recurrenceSettings []byte
	)

	if err := scanner.Scan(
		&template.ID,
		&template.Title,
		&template.Description,
		&template.Timezone,
		&startDate,
		&endDate,
		&recurrenceType,
		&recurrenceSettings,
		&template.IsActive,
		&template.CreatedAt,
		&template.UpdatedAt,
	); err != nil {
		return nil, err
	}

	template.StartDate = dateonly.FromTime(startDate)
	if endDate.Valid {
		parsed := dateonly.FromTime(endDate.Time)
		template.EndDate = &parsed
	}
	template.RecurrenceType = templatedomain.RecurrenceType(recurrenceType)
	if err := json.Unmarshal(recurrenceSettings, &template.RecurrenceSetting); err != nil {
		return nil, err
	}

	return &template, nil
}

func mustMarshalSettings(settings templatedomain.RecurrenceSettings) []byte {
	payload, err := json.Marshal(settings)
	if err != nil {
		panic(err)
	}

	return payload
}

func nullDate(value *dateonly.Date) any {
	if value == nil {
		return nil
	}

	return value.ToTime(time.UTC)
}
