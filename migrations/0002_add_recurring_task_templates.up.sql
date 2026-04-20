CREATE TABLE IF NOT EXISTS recurring_task_templates (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	timezone TEXT NOT NULL,
	start_date DATE NOT NULL,
	end_date DATE NULL,
	recurrence_type TEXT NOT NULL,
	recurrence_settings JSONB NOT NULL,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tasks
	ADD COLUMN IF NOT EXISTS template_id BIGINT NULL,
	ADD COLUMN IF NOT EXISTS scheduled_for DATE NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_template_scheduled_for_unique
	ON tasks (template_id, scheduled_for)
	WHERE template_id IS NOT NULL AND scheduled_for IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tasks_template_id ON tasks (template_id);
CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_for ON tasks (scheduled_for);
CREATE INDEX IF NOT EXISTS idx_recurring_task_templates_active ON recurring_task_templates (is_active);
