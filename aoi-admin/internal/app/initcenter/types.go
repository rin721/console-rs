package initcenter

import initdto "github.com/rei0721/go-scaffold/internal/app/initcenter/dto"

type Source = initdto.Source

const (
	SourceCLI = initdto.SourceCLI
	SourceWeb = initdto.SourceWeb
)

type Mode = initdto.Mode

const (
	ModeFirstRun = initdto.ModeFirstRun
	ModeRepair   = initdto.ModeRepair
	ModeVerify   = initdto.ModeVerify
)

type RunStatus = initdto.RunStatus

const (
	RunStatusPending   = initdto.RunStatusPending
	RunStatusRunning   = initdto.RunStatusRunning
	RunStatusSucceeded = initdto.RunStatusSucceeded
	RunStatusFailed    = initdto.RunStatusFailed
)

type StepStatus = initdto.StepStatus

const (
	StepStatusPending   = initdto.StepStatusPending
	StepStatusSkipped   = initdto.StepStatusSkipped
	StepStatusRunning   = initdto.StepStatusRunning
	StepStatusSucceeded = initdto.StepStatusSucceeded
	StepStatusFailed    = initdto.StepStatusFailed
)

type Input = initdto.Input
type Status = initdto.Status
type RunResult = initdto.RunResult
type RunReport = initdto.RunReport
type StepReport = initdto.StepReport
type SetupSchema = initdto.SetupSchema
type StepSchema = initdto.StepSchema
type FieldGroup = initdto.FieldGroup
type FieldSchema = initdto.FieldSchema
type VisibilityCondition = initdto.VisibilityCondition
type Option = initdto.Option
type TestResult = initdto.TestResult
type ConfigSaveResult = initdto.ConfigSaveResult
type InitReport = initdto.InitReport
type CompleteResult = initdto.CompleteResult
type RunRequest = initdto.RunRequest
type ConfigRequest = initdto.ConfigRequest
type TestRequest = initdto.TestRequest
type SkipRequest = initdto.SkipRequest
type CompleteRequest = initdto.CompleteRequest
