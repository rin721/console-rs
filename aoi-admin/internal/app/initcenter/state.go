package initcenter

import (
	"context"
	"errors"
	"strings"
	"time"

	dbpkg "github.com/rei0721/go-scaffold/pkg/database"
)

type runRecord struct {
	ID                int64      `gorm:"column:id;primaryKey"`
	RunKey            string     `gorm:"column:run_key"`
	Source            string     `gorm:"column:source"`
	Mode              string     `gorm:"column:mode"`
	Status            string     `gorm:"column:status"`
	CurrentStep       string     `gorm:"column:current_step"`
	StartedByUserID   int64      `gorm:"column:started_by_user_id"`
	IPAddress         string     `gorm:"column:ip_address"`
	UserAgent         string     `gorm:"column:user_agent"`
	ConfigPath        string     `gorm:"column:config_path"`
	ConfigFingerprint string     `gorm:"column:config_fingerprint"`
	LastError         string     `gorm:"column:last_error"`
	StartedAt         *time.Time `gorm:"column:started_at"`
	FinishedAt        *time.Time `gorm:"column:finished_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

func (runRecord) TableName() string { return "system_initialization_runs" }

type stepRecord struct {
	ID                   int64      `gorm:"column:id;primaryKey"`
	RunID                int64      `gorm:"column:run_id"`
	StepKey              string     `gorm:"column:step_key"`
	Phase                string     `gorm:"column:phase"`
	Order                int        `gorm:"column:step_order"`
	Status               string     `gorm:"column:status"`
	Required             bool       `gorm:"column:required"`
	Retryable            bool       `gorm:"column:retryable"`
	Idempotent           bool       `gorm:"column:idempotent"`
	Attempt              int        `gorm:"column:attempt"`
	InputSummary         string     `gorm:"column:input_summary"`
	OutputSummary        string     `gorm:"column:output_summary"`
	TestStatus           string     `gorm:"column:test_status"`
	TestSummary          string     `gorm:"column:test_summary"`
	TestError            string     `gorm:"column:test_error"`
	TestInputFingerprint string     `gorm:"column:test_input_fingerprint"`
	SkippedReason        string     `gorm:"column:skipped_reason"`
	RepairHint           string     `gorm:"column:repair_hint"`
	RestartRequired      bool       `gorm:"column:restart_required"`
	ErrorCode            string     `gorm:"column:error_code"`
	ErrorMessage         string     `gorm:"column:error_message"`
	StartedAt            *time.Time `gorm:"column:started_at"`
	FinishedAt           *time.Time `gorm:"column:finished_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
}

func (stepRecord) TableName() string { return "system_initialization_steps" }

type stateStore struct {
	db  dbpkg.Database
	ids interface {
		NextID() int64
	}
}

func newStateStore(db dbpkg.Database, ids interface{ NextID() int64 }) *stateStore {
	if db == nil || ids == nil {
		return nil
	}
	return &stateStore{db: db, ids: ids}
}

func (s *stateStore) ensure(ctx context.Context) error {
	if s == nil {
		return nil
	}
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS system_initialization_runs (
			id BIGINT PRIMARY KEY,
			run_key VARCHAR(64) NOT NULL UNIQUE,
			source VARCHAR(32) NOT NULL,
			mode VARCHAR(32) NOT NULL,
			status VARCHAR(32) NOT NULL,
			current_step VARCHAR(128) NOT NULL DEFAULT '',
			started_by_user_id BIGINT NOT NULL DEFAULT 0,
			ip_address VARCHAR(64) NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL,
			config_path VARCHAR(512) NOT NULL DEFAULT '',
			config_fingerprint VARCHAR(128) NOT NULL DEFAULT '',
			last_error TEXT NOT NULL,
			started_at TIMESTAMP NULL,
			finished_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS system_initialization_steps (
			id BIGINT PRIMARY KEY,
			run_id BIGINT NOT NULL,
			step_key VARCHAR(128) NOT NULL,
			phase VARCHAR(64) NOT NULL,
			step_order INTEGER NOT NULL,
			status VARCHAR(32) NOT NULL,
			required BOOLEAN NOT NULL DEFAULT true,
			retryable BOOLEAN NOT NULL DEFAULT true,
			idempotent BOOLEAN NOT NULL DEFAULT true,
			attempt INTEGER NOT NULL DEFAULT 0,
			input_summary TEXT NOT NULL,
			output_summary TEXT NOT NULL,
			test_status VARCHAR(32) NOT NULL DEFAULT '',
			test_summary TEXT NOT NULL DEFAULT '',
			test_error TEXT NOT NULL DEFAULT '',
			test_input_fingerprint VARCHAR(128) NOT NULL DEFAULT '',
			skipped_reason TEXT NOT NULL DEFAULT '',
			repair_hint TEXT NOT NULL DEFAULT '',
			restart_required BOOLEAN NOT NULL DEFAULT false,
			error_code VARCHAR(128) NOT NULL DEFAULT '',
			error_message TEXT NOT NULL,
			started_at TIMESTAMP NULL,
			finished_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			UNIQUE (run_id, step_key)
		)`,
	} {
		if _, err := s.db.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	for _, stmt := range []string{
		`ALTER TABLE system_initialization_steps ADD COLUMN test_input_fingerprint VARCHAR(128) NOT NULL DEFAULT ''`,
	} {
		if _, err := s.db.Exec(ctx, stmt); err != nil && !isDuplicateColumnError(err) {
			return err
		}
	}
	return nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column") ||
		strings.Contains(message, "already exists") ||
		strings.Contains(message, "duplicate column name")
}

func (s *stateStore) latestOrCreateRun(ctx context.Context, input Input, configPath string, fingerprint string) (*runRecord, error) {
	if s == nil {
		return nil, nil
	}
	run, _, err := s.latestRun(ctx)
	if err != nil {
		return nil, err
	}
	if run != nil && run.Status != string(RunStatusSucceeded) {
		return run, nil
	}
	return s.createRun(ctx, input, configPath, fingerprint)
}

func (s *stateStore) createRun(ctx context.Context, input Input, configPath string, fingerprint string) (*runRecord, error) {
	if s == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	run := &runRecord{
		ID:                s.ids.NextID(),
		RunKey:            time.Now().UTC().Format("20060102150405") + "-" + shortID(s.ids.NextID()),
		Source:            string(input.Source),
		Mode:              string(defaultMode(input.Mode)),
		Status:            string(RunStatusRunning),
		IPAddress:         input.IPAddress,
		UserAgent:         input.UserAgent,
		ConfigPath:        configPath,
		ConfigFingerprint: fingerprint,
		StartedAt:         &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.db.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *stateStore) saveRun(ctx context.Context, run *runRecord) error {
	if s == nil || run == nil {
		return nil
	}
	run.UpdatedAt = time.Now().UTC()
	return s.db.Save(ctx, run)
}

func (s *stateStore) upsertStep(ctx context.Context, run *runRecord, def stepDefinition, status StepStatus, attemptDelta int, output string, err error) (*stepRecord, error) {
	if s == nil || run == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	var step stepRecord
	findErr := s.db.First(ctx, &step, dbpkg.Where("run_id = ?", run.ID), dbpkg.Where("step_key = ?", def.Key))
	if findErr != nil && !errors.Is(findErr, dbpkg.ErrNotFound) {
		return nil, findErr
	}
	if errors.Is(findErr, dbpkg.ErrNotFound) {
		step = stepRecord{
			ID:           s.ids.NextID(),
			RunID:        run.ID,
			StepKey:      def.Key,
			Phase:        def.Phase,
			Order:        def.Order,
			Required:     def.Required,
			Retryable:    def.Retryable,
			Idempotent:   def.Idempotent,
			InputSummary: inputSummary(def.Key),
			CreatedAt:    now,
		}
	}
	step.Status = string(status)
	step.Attempt += attemptDelta
	step.OutputSummary = output
	step.ErrorCode = ""
	step.ErrorMessage = ""
	if status == StepStatusRunning && step.StartedAt == nil {
		step.StartedAt = &now
	}
	if status == StepStatusSucceeded || status == StepStatusSkipped || status == StepStatusFailed {
		step.FinishedAt = &now
	}
	if err != nil {
		step.ErrorCode = "initialization_step_failed"
		step.ErrorMessage = err.Error()
	}
	step.UpdatedAt = now
	if errors.Is(findErr, dbpkg.ErrNotFound) {
		return &step, s.db.Create(ctx, &step)
	}
	return &step, s.db.Save(ctx, &step)
}

func (s *stateStore) updateStepTest(ctx context.Context, run *runRecord, def stepDefinition, test TestResult) error {
	if s == nil || run == nil {
		return nil
	}
	step, err := s.upsertStep(ctx, run, def, StepStatusPending, 0, "", nil)
	if err != nil {
		return err
	}
	step.TestStatus = test.Status
	step.TestSummary = test.Summary
	step.TestError = test.Error
	step.TestInputFingerprint = test.InputFingerprint
	step.RepairHint = test.RepairHint
	step.RestartRequired = test.RestartRequired
	step.UpdatedAt = time.Now().UTC()
	return s.db.Save(ctx, step)
}

func (s *stateStore) skipStep(ctx context.Context, run *runRecord, def stepDefinition, reason string) error {
	if s == nil || run == nil {
		return nil
	}
	step, err := s.upsertStep(ctx, run, def, StepStatusSkipped, 0, "skipped by operator", nil)
	if err != nil {
		return err
	}
	step.SkippedReason = reason
	step.UpdatedAt = time.Now().UTC()
	return s.db.Save(ctx, step)
}

func (s *stateStore) latestRun(ctx context.Context) (*runRecord, []stepRecord, error) {
	if s == nil {
		return nil, nil, nil
	}
	if err := s.ensure(ctx); err != nil {
		return nil, nil, err
	}
	var run runRecord
	if err := s.db.First(ctx, &run, dbpkg.Order("created_at DESC, id DESC")); err != nil {
		if errors.Is(err, dbpkg.ErrNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	steps, err := s.steps(ctx, run.ID)
	if err != nil {
		return nil, nil, err
	}
	return &run, steps, nil
}

func (s *stateStore) findRun(ctx context.Context, key string) (*runRecord, []stepRecord, error) {
	if s == nil {
		return nil, nil, nil
	}
	if strings.TrimSpace(key) == "" {
		return nil, nil, ErrInitializationRunAbsent
	}
	var run runRecord
	if err := s.db.First(ctx, &run, dbpkg.Where("run_key = ?", key)); err != nil {
		if errors.Is(err, dbpkg.ErrNotFound) {
			return nil, nil, ErrInitializationRunAbsent
		}
		return nil, nil, err
	}
	steps, err := s.steps(ctx, run.ID)
	return &run, steps, err
}

func (s *stateStore) requireRun(ctx context.Context, key string) error {
	if s == nil {
		return nil
	}
	if err := s.ensure(ctx); err != nil {
		return err
	}
	_, _, err := s.findRun(ctx, key)
	return err
}

func (s *stateStore) steps(ctx context.Context, runID int64) ([]stepRecord, error) {
	var steps []stepRecord
	if err := s.db.Find(ctx, &steps, dbpkg.Where("run_id = ?", runID), dbpkg.Order("step_order ASC, id ASC")); err != nil {
		return nil, err
	}
	return steps, nil
}

func runReport(run *runRecord) RunReport {
	if run == nil {
		return RunReport{}
	}
	return RunReport{
		ID:                run.RunKey,
		Source:            Source(run.Source),
		Mode:              Mode(run.Mode),
		Status:            RunStatus(run.Status),
		CurrentStep:       run.CurrentStep,
		ConfigPath:        run.ConfigPath,
		ConfigFingerprint: run.ConfigFingerprint,
		LastError:         run.LastError,
		StartedAt:         run.StartedAt,
		FinishedAt:        run.FinishedAt,
		CreatedAt:         run.CreatedAt,
		UpdatedAt:         run.UpdatedAt,
	}
}
