package tasktemplate

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/taskservice/internal/domain/dateonly"
	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/tasktemplate"
)

func TestCreateRollsBackWhenGenerationFails(t *testing.T) {
	t.Parallel()

	store := newTemplateStore()
	service := NewService(store, store, store)
	service.now = func() time.Time { return time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC) }
	store.failCreateGenerated = errors.New("generation failed")

	_, err := service.Create(context.Background(), CreateInput{
		Title:          "Daily calls",
		Timezone:       "UTC",
		StartDate:      "2026-04-20",
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSettings: RecurrenceSettingsInput{
			EveryNDays: 1,
		},
	})

	if err == nil {
		t.Fatal("expected create error")
	}
	if len(store.templates) != 0 {
		t.Fatalf("expected transaction rollback, got %d templates", len(store.templates))
	}
	if len(store.generatedTasks) != 0 {
		t.Fatalf("expected no generated tasks, got %d", len(store.generatedTasks))
	}
}

func TestUpdateRollsBackWhenGenerationFails(t *testing.T) {
	t.Parallel()

	store := newTemplateStore()
	service := NewService(store, store, store)
	service.now = func() time.Time { return time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC) }

	template := templatedomain.Template{
		ID:             1,
		Title:          "Old title",
		Description:    "Old desc",
		Timezone:       "UTC",
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			EveryNDays: 1,
		},
		IsActive: true,
	}
	store.templates[template.ID] = template
	store.nextTemplateID = 2
	store.generatedTasks = []taskdomain.Task{
		newGeneratedTask(10, 1, "2026-04-20", taskdomain.StatusInProgress),
		newGeneratedTask(11, 1, "2026-04-21", taskdomain.StatusNew),
	}
	store.failCreateGenerated = errors.New("generation failed")

	_, err := service.Update(context.Background(), 1, UpdateInput{
		Title:          "New title",
		Description:    "New desc",
		Timezone:       "UTC",
		StartDate:      "2026-04-20",
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSettings: RecurrenceSettingsInput{
			EveryNDays: 2,
		},
	})

	if err == nil {
		t.Fatal("expected update error")
	}

	if got := store.templates[1].Title; got != "Old title" {
		t.Fatalf("expected template rollback, got title %q", got)
	}
	if len(store.generatedTasks) != 2 {
		t.Fatalf("expected generated tasks rollback, got %d tasks", len(store.generatedTasks))
	}
	if store.generatedTasks[1].ScheduledFor == nil || dateonly.FromTime(*store.generatedTasks[1].ScheduledFor).String() != "2026-04-21" {
		t.Fatal("expected future generated task to be preserved after rollback")
	}
}

func TestUpdateKeepsTodaysGeneratedTask(t *testing.T) {
	t.Parallel()

	store := newTemplateStore()
	service := NewService(store, store, store)
	service.now = func() time.Time { return time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC) }

	template := templatedomain.Template{
		ID:             1,
		Title:          "Calls",
		Timezone:       "UTC",
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			EveryNDays: 1,
		},
		IsActive: true,
	}
	store.templates[template.ID] = template
	store.nextTemplateID = 2
	store.generatedTasks = []taskdomain.Task{
		newGeneratedTask(10, 1, "2026-04-20", taskdomain.StatusDone),
		newGeneratedTask(11, 1, "2026-04-21", taskdomain.StatusNew),
	}

	updated, err := service.Update(context.Background(), 1, UpdateInput{
		Title:          "Calls",
		Timezone:       "UTC",
		StartDate:      "2026-04-20",
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSettings: RecurrenceSettingsInput{
			EveryNDays: 2,
		},
	})

	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated template")
	}

	foundToday := false
	for _, task := range store.generatedTasks {
		if task.ScheduledFor != nil && dateonly.FromTime(*task.ScheduledFor).String() == "2026-04-20" && task.Status == taskdomain.StatusDone {
			foundToday = true
		}
		if task.ScheduledFor != nil && dateonly.FromTime(*task.ScheduledFor).String() == "2026-04-21" && task.Status == taskdomain.StatusNew {
			t.Fatal("expected old future task to be removed on successful update")
		}
	}
	if !foundToday {
		t.Fatal("expected today's generated task to be preserved")
	}
}

func TestSyncGeneratedTasksContinuesAfterTemplateError(t *testing.T) {
	t.Parallel()

	store := newTemplateStore()
	service := NewService(store, store, store)
	service.now = func() time.Time { return time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC) }

	brokenTemplate := templatedomain.Template{
		ID:             1,
		Title:          "Broken template",
		Timezone:       "UTC",
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			EveryNDays: 1,
		},
		IsActive: true,
	}
	healthyTemplate := templatedomain.Template{
		ID:             2,
		Title:          "Healthy template",
		Timezone:       "UTC",
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			EveryNDays: 1,
		},
		IsActive: true,
	}

	store.templates[brokenTemplate.ID] = brokenTemplate
	store.templates[healthyTemplate.ID] = healthyTemplate
	store.failCreateGeneratedForTemplateIDs = map[int64]error{
		1: errors.New("broken template generation"),
	}

	err := service.SyncGeneratedTasks(context.Background())
	if err == nil {
		t.Fatal("expected sync error")
	}
	if !strings.Contains(err.Error(), "sync template 1") {
		t.Fatalf("expected aggregated error to mention broken template, got %v", err)
	}

	foundHealthyTask := false
	for _, task := range store.generatedTasks {
		if task.TemplateID != nil && *task.TemplateID == 1 {
			t.Fatal("did not expect generated tasks for broken template")
		}
		if task.TemplateID != nil && *task.TemplateID == 2 {
			foundHealthyTask = true
		}
	}

	if !foundHealthyTask {
		t.Fatal("expected healthy template to continue generating tasks")
	}
}

func TestDeleteDeactivatesTemplateAndRemovesOnlyFutureNewGeneratedTasks(t *testing.T) {
	t.Parallel()

	store := newTemplateStore()
	service := NewService(store, store, store)
	service.now = func() time.Time { return time.Date(2026, 4, 15, 8, 0, 0, 0, time.UTC) }

	template := templatedomain.Template{
		ID:             1,
		Title:          "Every second day",
		Timezone:       "UTC",
		StartDate:      mustDate(t, "2026-04-01"),
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			EveryNDays: 2,
		},
		IsActive: true,
	}
	store.templates[template.ID] = template
	store.generatedTasks = []taskdomain.Task{
		newGeneratedTask(10, 1, "2026-04-14", taskdomain.StatusDone),
		newGeneratedTask(11, 1, "2026-04-15", taskdomain.StatusNew),
		newGeneratedTask(12, 1, "2026-04-16", taskdomain.StatusNew),
		newGeneratedTask(13, 1, "2026-04-18", taskdomain.StatusInProgress),
		newGeneratedTask(14, 1, "2026-04-20", taskdomain.StatusNew),
	}

	err := service.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if store.templates[1].IsActive {
		t.Fatal("expected template to be deactivated")
	}

	remainingDates := make(map[string]taskdomain.Status, len(store.generatedTasks))
	for _, task := range store.generatedTasks {
		if task.ScheduledFor == nil {
			continue
		}
		remainingDates[dateonly.FromTime(*task.ScheduledFor).String()] = task.Status
	}

	if _, exists := remainingDates["2026-04-16"]; exists {
		t.Fatal("expected future new task on 2026-04-16 to be removed")
	}
	if _, exists := remainingDates["2026-04-20"]; exists {
		t.Fatal("expected future new task on 2026-04-20 to be removed")
	}
	if status, exists := remainingDates["2026-04-14"]; !exists || status != taskdomain.StatusDone {
		t.Fatal("expected past completed task to remain")
	}
	if status, exists := remainingDates["2026-04-15"]; !exists || status != taskdomain.StatusNew {
		t.Fatal("expected current-day task to remain")
	}
	if status, exists := remainingDates["2026-04-18"]; !exists || status != taskdomain.StatusInProgress {
		t.Fatal("expected future in-progress task to remain")
	}
}

func TestValidateInputRejectsInvalidTimezone(t *testing.T) {
	t.Parallel()

	_, err := validateInput(CreateInput{
		Title:          "Daily calls",
		Timezone:       "Mars/Olympus",
		StartDate:      "2026-04-20",
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSettings: RecurrenceSettingsInput{
			EveryNDays: 1,
		},
	})

	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateInputRejectsIrrelevantFields(t *testing.T) {
	t.Parallel()

	_, err := validateInput(CreateInput{
		Title:          "Daily calls",
		Timezone:       "UTC",
		StartDate:      "2026-04-20",
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSettings: RecurrenceSettingsInput{
			EveryNDays: 1,
			Weekdays:   []templatedomain.Weekday{templatedomain.WeekdayMonday},
		},
	})

	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBuildOccurrencesDaily(t *testing.T) {
	t.Parallel()

	location := time.UTC
	template := templatedomain.Template{
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceDaily,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			EveryNDays: 2,
		},
	}

	got := buildOccurrences(template, mustDate(t, "2026-04-20"), mustDate(t, "2026-04-26"), location)

	assertDates(t, got, "2026-04-20", "2026-04-22", "2026-04-24", "2026-04-26")
}

func TestBuildOccurrencesMonthlyLastDayStrategy(t *testing.T) {
	t.Parallel()

	location := time.UTC
	template := templatedomain.Template{
		StartDate:      mustDate(t, "2026-01-31"),
		RecurrenceType: templatedomain.RecurrenceMonthly,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			DayOfMonth:         31,
			ShortMonthStrategy: templatedomain.ShortMonthLastDay,
		},
	}

	got := buildOccurrences(template, mustDate(t, "2026-01-31"), mustDate(t, "2026-03-31"), location)

	assertDates(t, got, "2026-01-31", "2026-02-28", "2026-03-31")
}

func TestBuildOccurrencesSpecificDates(t *testing.T) {
	t.Parallel()

	template := templatedomain.Template{
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceSpecificDates,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			SpecificDates: []dateonly.Date{
				mustDate(t, "2026-04-19"),
				mustDate(t, "2026-04-20"),
				mustDate(t, "2026-05-01"),
			},
		},
	}

	got := buildOccurrences(template, mustDate(t, "2026-04-20"), mustDate(t, "2026-04-30"), time.UTC)

	assertDates(t, got, "2026-04-20")
}

func TestBuildOccurrencesEvenDays(t *testing.T) {
	t.Parallel()

	location := time.UTC
	template := templatedomain.Template{
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceEvenOdd,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			EvenOddMode: templatedomain.EvenOddEven,
		},
	}

	got := buildOccurrences(template, mustDate(t, "2026-04-20"), mustDate(t, "2026-04-25"), location)

	assertDates(t, got, "2026-04-20", "2026-04-22", "2026-04-24")
}

func TestBuildOccurrencesWeekdays(t *testing.T) {
	t.Parallel()

	location := time.UTC
	template := templatedomain.Template{
		StartDate:      mustDate(t, "2026-04-20"),
		RecurrenceType: templatedomain.RecurrenceWeekdays,
		RecurrenceSetting: templatedomain.RecurrenceSettings{
			Weekdays: []templatedomain.Weekday{
				templatedomain.WeekdayTuesday,
				templatedomain.WeekdayThursday,
			},
		},
	}

	got := buildOccurrences(template, mustDate(t, "2026-04-20"), mustDate(t, "2026-04-30"), location)

	assertDates(t, got, "2026-04-21", "2026-04-23", "2026-04-28", "2026-04-30")
}

func mustDate(t *testing.T, value string) dateonly.Date {
	t.Helper()

	date, err := dateonly.Parse(value)
	if err != nil {
		t.Fatalf("parse date %s: %v", value, err)
	}

	return date
}

func assertDates(t *testing.T, got []dateonly.Date, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unexpected number of dates: got=%d want=%d", len(got), len(want))
	}

	for index := range want {
		if got[index].String() != want[index] {
			t.Fatalf("unexpected date at index %d: got=%s want=%s", index, got[index].String(), want[index])
		}
	}
}

type templateStore struct {
	nextTemplateID                    int64
	nextTaskID                        int64
	templates                         map[int64]templatedomain.Template
	generatedTasks                    []taskdomain.Task
	failCreateGenerated               error
	failCreateGeneratedForTemplateIDs map[int64]error
}

func newTemplateStore() *templateStore {
	return &templateStore{
		nextTemplateID: 1,
		nextTaskID:     100,
		templates:      make(map[int64]templatedomain.Template),
	}
}

func (s *templateStore) WithinTransaction(_ context.Context, fn func(TemplateRepository, GeneratedTaskRepository) error) error {
	clone := s.clone()
	if err := fn(clone, clone); err != nil {
		return err
	}

	s.nextTemplateID = clone.nextTemplateID
	s.nextTaskID = clone.nextTaskID
	s.templates = clone.templates
	s.generatedTasks = clone.generatedTasks
	return nil
}

func (s *templateStore) Create(_ context.Context, template *templatedomain.Template) (*templatedomain.Template, error) {
	created := *template
	created.ID = s.nextTemplateID
	s.nextTemplateID++
	s.templates[created.ID] = created
	return cloneTemplate(created), nil
}

func (s *templateStore) GetByID(_ context.Context, id int64) (*templatedomain.Template, error) {
	template, ok := s.templates[id]
	if !ok || !template.IsActive {
		return nil, templatedomain.ErrNotFound
	}
	return cloneTemplate(template), nil
}

func (s *templateStore) Update(_ context.Context, template *templatedomain.Template) (*templatedomain.Template, error) {
	current, ok := s.templates[template.ID]
	if !ok || !current.IsActive {
		return nil, templatedomain.ErrNotFound
	}

	updated := *template
	updated.CreatedAt = current.CreatedAt
	s.templates[updated.ID] = updated
	return cloneTemplate(updated), nil
}

func (s *templateStore) Deactivate(_ context.Context, id int64, updatedAt time.Time) error {
	template, ok := s.templates[id]
	if !ok || !template.IsActive {
		return templatedomain.ErrNotFound
	}
	template.IsActive = false
	template.UpdatedAt = updatedAt
	s.templates[id] = template
	return nil
}

func (s *templateStore) List(_ context.Context) ([]templatedomain.Template, error) {
	result := make([]templatedomain.Template, 0, len(s.templates))
	for _, template := range s.templates {
		if template.IsActive {
			result = append(result, template)
		}
	}
	return result, nil
}

func (s *templateStore) ListActive(ctx context.Context) ([]templatedomain.Template, error) {
	return s.List(ctx)
}

func (s *templateStore) CreateGenerated(_ context.Context, task *taskdomain.Task) error {
	if task.TemplateID != nil && s.failCreateGeneratedForTemplateIDs != nil {
		if err, exists := s.failCreateGeneratedForTemplateIDs[*task.TemplateID]; exists {
			return err
		}
	}

	if s.failCreateGenerated != nil {
		return s.failCreateGenerated
	}

	for _, existing := range s.generatedTasks {
		if existing.TemplateID != nil && task.TemplateID != nil && *existing.TemplateID == *task.TemplateID &&
			existing.ScheduledFor != nil && task.ScheduledFor != nil &&
			dateonly.FromTime(*existing.ScheduledFor).Equal(dateonly.FromTime(*task.ScheduledFor)) {
			return nil
		}
	}

	created := *task
	created.ID = s.nextTaskID
	s.nextTaskID++
	s.generatedTasks = append(s.generatedTasks, created)
	return nil
}

func (s *templateStore) DeleteFutureByTemplate(_ context.Context, templateID int64, fromDate dateonly.Date) error {
	filtered := s.generatedTasks[:0]
	for _, task := range s.generatedTasks {
		if task.TemplateID == nil || *task.TemplateID != templateID || task.ScheduledFor == nil {
			filtered = append(filtered, task)
			continue
		}

		scheduledFor := dateonly.FromTime(*task.ScheduledFor)
		if scheduledFor.After(fromDate) && task.Status == taskdomain.StatusNew {
			continue
		}

		filtered = append(filtered, task)
	}
	s.generatedTasks = filtered
	return nil
}

func (s *templateStore) clone() *templateStore {
	templates := make(map[int64]templatedomain.Template, len(s.templates))
	for id, template := range s.templates {
		templates[id] = *cloneTemplate(template)
	}

	tasks := make([]taskdomain.Task, 0, len(s.generatedTasks))
	for _, task := range s.generatedTasks {
		tasks = append(tasks, cloneTask(task))
	}

	return &templateStore{
		nextTemplateID:                    s.nextTemplateID,
		nextTaskID:                        s.nextTaskID,
		templates:                         templates,
		generatedTasks:                    tasks,
		failCreateGenerated:               s.failCreateGenerated,
		failCreateGeneratedForTemplateIDs: mapsClone(s.failCreateGeneratedForTemplateIDs),
	}
}

func mapsClone(src map[int64]error) map[int64]error {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[int64]error, len(src))
	for key, value := range src {
		dst[key] = value
	}

	return dst
}

func cloneTemplate(template templatedomain.Template) *templatedomain.Template {
	cloned := template
	if template.EndDate != nil {
		endDate := *template.EndDate
		cloned.EndDate = &endDate
	}
	if len(template.RecurrenceSetting.SpecificDates) > 0 {
		cloned.RecurrenceSetting.SpecificDates = append([]dateonly.Date(nil), template.RecurrenceSetting.SpecificDates...)
	}
	if len(template.RecurrenceSetting.Weekdays) > 0 {
		cloned.RecurrenceSetting.Weekdays = append([]templatedomain.Weekday(nil), template.RecurrenceSetting.Weekdays...)
	}
	return &cloned
}

func cloneTask(task taskdomain.Task) taskdomain.Task {
	cloned := task
	if task.TemplateID != nil {
		templateID := *task.TemplateID
		cloned.TemplateID = &templateID
	}
	if task.ScheduledFor != nil {
		scheduledFor := *task.ScheduledFor
		cloned.ScheduledFor = &scheduledFor
	}
	return cloned
}

func newGeneratedTask(id, templateID int64, scheduledFor string, status taskdomain.Status) taskdomain.Task {
	templateRef := templateID
	scheduledAt := mustStaticDate(scheduledFor).ToTime(time.UTC)
	return taskdomain.Task{
		ID:           id,
		Title:        "generated",
		Status:       status,
		TemplateID:   &templateRef,
		ScheduledFor: &scheduledAt,
	}
}

func mustStaticDate(value string) dateonly.Date {
	date, err := dateonly.Parse(value)
	if err != nil {
		panic(err)
	}
	return date
}
