package dto

import (
	"time"

	appconfig "github.com/rei0721/go-scaffold/internal/config"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
)

type Source string

const (
	SourceCLI Source = "cli"
	SourceWeb Source = "web"
)

type Mode string

const (
	ModeFirstRun Mode = "first_run"
	ModeRepair   Mode = "repair"
	ModeVerify   Mode = "verify"
)

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusSucceeded RunStatus = "succeeded"
	RunStatusFailed    RunStatus = "failed"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusRunning   StepStatus = "running"
	StepStatusSucceeded StepStatus = "succeeded"
	StepStatusFailed    StepStatus = "failed"
)

type Input struct {
	Source             Source
	Mode               Mode
	SetupToken         string
	OrgCode            string
	OrgName            string
	AdminUsername      string
	AdminEmail         string
	AdminDisplayName   string
	AdminPassword      string
	UserAgent          string
	IPAddress          string
	ProductCode        string
	ClientType         string
	IssueLoginTokens   bool
	CreateServiceToken bool
	ServiceTokenDays   int
	ServiceTokenRemark string
}

type Status struct {
	Required        bool                         `json:"required"`
	Completed       bool                         `json:"completed"`
	CurrentStep     string                       `json:"currentStep"`
	AllowedActions  []string                     `json:"allowedActions"`
	Diagnostics     []appconfig.ConfigDiagnostic `json:"diagnostics"`
	RestartRequired bool                         `json:"restartRequired"`
	RestartReason   string                       `json:"restartReason,omitempty"`
	PasswordPolicy  iamservice.PasswordPolicy    `json:"passwordPolicy"`
	Steps           []StepReport                 `json:"steps"`
	LastRun         *RunReport                   `json:"lastRun,omitempty"`
	Report          *InitReport                  `json:"report,omitempty"`
}

type RunResult struct {
	Run                RunReport                       `json:"run"`
	LoginTokens        iamservice.SessionSnapshot      `json:"loginTokens,omitempty"`
	LoginTokensIssued  bool                            `json:"loginTokensIssued"`
	ServiceToken       iamservice.CreateAPITokenResult `json:"serviceToken,omitempty"`
	ServiceTokenIssued bool                            `json:"serviceTokenIssued"`
	RestartRequired    bool                            `json:"restartRequired"`
	RestartReason      string                          `json:"restartReason,omitempty"`
	Report             InitReport                      `json:"report"`
	Steps              []StepReport                    `json:"steps"`
	RawLoginTokens     iamservice.TokenPair            `json:"-"`
}

type RunReport struct {
	ID                string     `json:"id"`
	Source            Source     `json:"source"`
	Mode              Mode       `json:"mode"`
	Status            RunStatus  `json:"status"`
	CurrentStep       string     `json:"currentStep"`
	ConfigPath        string     `json:"configPath"`
	ConfigFingerprint string     `json:"configFingerprint"`
	LastError         string     `json:"lastError"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	FinishedAt        *time.Time `json:"finishedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type StepReport struct {
	Key                  string         `json:"key"`
	Phase                string         `json:"phase"`
	Order                int            `json:"order"`
	Title                string         `json:"title"`
	TitleKey             string         `json:"titleKey,omitempty"`
	Goal                 string         `json:"goal"`
	GoalKey              string         `json:"goalKey,omitempty"`
	Prerequisites        []string       `json:"prerequisites"`
	PrerequisiteKeys     []string       `json:"prerequisiteKeys,omitempty"`
	UserInputs           []string       `json:"userInputs"`
	UserInputKeys        []string       `json:"userInputKeys,omitempty"`
	AutomaticActions     []string       `json:"automaticActions"`
	AutomaticActionKeys  []string       `json:"automaticActionKeys,omitempty"`
	CompletionMark       string         `json:"completionMark"`
	CompletionMarkKey    string         `json:"completionMarkKey,omitempty"`
	Recovery             string         `json:"recovery"`
	RecoveryKey          string         `json:"recoveryKey,omitempty"`
	Required             bool           `json:"required"`
	Retryable            bool           `json:"retryable"`
	Idempotent           bool           `json:"idempotent"`
	Skippable            bool           `json:"skippable"`
	Testable             bool           `json:"testable"`
	Dependencies         []string       `json:"dependencies"`
	Schema               StepSchema     `json:"schema"`
	Status               StepStatus     `json:"status"`
	Attempt              int            `json:"attempt"`
	OutputSummary        string         `json:"outputSummary"`
	OutputSummaryKey     string         `json:"outputSummaryKey,omitempty"`
	OutputSummaryArgs    map[string]any `json:"outputSummaryArgs,omitempty"`
	ErrorCode            string         `json:"errorCode"`
	ErrorMessage         string         `json:"errorMessage"`
	ErrorMessageKey      string         `json:"errorMessageKey,omitempty"`
	ErrorMessageArgs     map[string]any `json:"errorMessageArgs,omitempty"`
	RepairHint           string         `json:"repairHint"`
	RepairHintKey        string         `json:"repairHintKey,omitempty"`
	RepairHintArgs       map[string]any `json:"repairHintArgs,omitempty"`
	SkippedReason        string         `json:"skippedReason"`
	SkippedReasonKey     string         `json:"skippedReasonKey,omitempty"`
	SkippedReasonArgs    map[string]any `json:"skippedReasonArgs,omitempty"`
	TestStatus           string         `json:"testStatus"`
	TestSummary          string         `json:"testSummary"`
	TestSummaryKey       string         `json:"testSummaryKey,omitempty"`
	TestSummaryArgs      map[string]any `json:"testSummaryArgs,omitempty"`
	TestError            string         `json:"testError"`
	TestErrorKey         string         `json:"testErrorKey,omitempty"`
	TestErrorArgs        map[string]any `json:"testErrorArgs,omitempty"`
	TestInputFingerprint string         `json:"testInputFingerprint,omitempty"`
	RestartRequired      bool           `json:"restartRequired"`
	StartedAt            *time.Time     `json:"startedAt,omitempty"`
	FinishedAt           *time.Time     `json:"finishedAt,omitempty"`
}

type SetupSchema struct {
	Steps []StepSchema `json:"steps"`
}

type StepSchema struct {
	Key              string        `json:"key"`
	RouteSlug        string        `json:"routeSlug"`
	Phase            string        `json:"phase"`
	Order            int           `json:"order"`
	Title            string        `json:"title"`
	TitleKey         string        `json:"titleKey,omitempty"`
	Description      string        `json:"description"`
	DescriptionKey   string        `json:"descriptionKey,omitempty"`
	Required         bool          `json:"required"`
	Skippable        bool          `json:"skippable"`
	Testable         bool          `json:"testable"`
	Dependencies     []string      `json:"dependencies"`
	InputFingerprint string        `json:"inputFingerprint,omitempty"`
	Fields           []FieldSchema `json:"fields"`
	Groups           []FieldGroup  `json:"groups"`
}

type FieldGroup struct {
	Key            string               `json:"key"`
	Title          string               `json:"title"`
	TitleKey       string               `json:"titleKey,omitempty"`
	Description    string               `json:"description,omitempty"`
	DescriptionKey string               `json:"descriptionKey,omitempty"`
	VisibleWhen    *VisibilityCondition `json:"visibleWhen,omitempty"`
	Fields         []FieldSchema        `json:"fields"`
}

type FieldSchema struct {
	Key         string               `json:"key"`
	Label       string               `json:"label"`
	LabelKey    string               `json:"labelKey,omitempty"`
	Type        string               `json:"type"`
	Required    bool                 `json:"required"`
	Sensitive   bool                 `json:"sensitive"`
	Placeholder string               `json:"placeholder,omitempty"`
	Help        string               `json:"help,omitempty"`
	HelpKey     string               `json:"helpKey,omitempty"`
	Options     []Option             `json:"options,omitempty"`
	Default     any                  `json:"default,omitempty"`
	Value       any                  `json:"value,omitempty"`
	ConfigPath  string               `json:"configPath,omitempty"`
	VisibleWhen *VisibilityCondition `json:"visibleWhen,omitempty"`
}

type VisibilityCondition struct {
	Field  string   `json:"field"`
	Equals any      `json:"equals,omitempty"`
	In     []string `json:"in,omitempty"`
}

type Option struct {
	Label    string `json:"label"`
	LabelKey string `json:"labelKey,omitempty"`
	Value    string `json:"value"`
}

type TestResult struct {
	StepKey          string         `json:"stepKey"`
	Status           string         `json:"status"`
	Summary          string         `json:"summary"`
	SummaryKey       string         `json:"summaryKey,omitempty"`
	SummaryArgs      map[string]any `json:"summaryArgs,omitempty"`
	Error            string         `json:"error,omitempty"`
	ErrorKey         string         `json:"errorKey,omitempty"`
	ErrorArgs        map[string]any `json:"errorArgs,omitempty"`
	RepairHint       string         `json:"repairHint,omitempty"`
	RepairHintKey    string         `json:"repairHintKey,omitempty"`
	RepairHintArgs   map[string]any `json:"repairHintArgs,omitempty"`
	InputFingerprint string         `json:"inputFingerprint,omitempty"`
	RestartRequired  bool           `json:"restartRequired"`
	TestedAt         time.Time      `json:"testedAt"`
}

type ConfigSaveResult struct {
	StepKey                    string       `json:"stepKey"`
	InputSummary               string       `json:"inputSummary"`
	InputFingerprint           string       `json:"inputFingerprint,omitempty"`
	RestartRequired            bool         `json:"restartRequired"`
	RestartReason              string       `json:"restartReason,omitempty"`
	NextAction                 string       `json:"nextAction,omitempty"`
	EnvManagedPathsOverwritten []string     `json:"envManagedPathsOverwritten,omitempty"`
	EnvManagedPersistence      string       `json:"envManagedPersistence,omitempty"`
	Test                       *TestResult  `json:"test,omitempty"`
	Steps                      []StepReport `json:"steps"`
}

type InitReport struct {
	GeneratedAt     time.Time      `json:"generatedAt"`
	Successful      int            `json:"successful"`
	Failed          int            `json:"failed"`
	Skipped         int            `json:"skipped"`
	Risk            int            `json:"risk"`
	RestartRequired bool           `json:"restartRequired"`
	RestartReason   string         `json:"restartReason,omitempty"`
	Summary         string         `json:"summary"`
	SummaryKey      string         `json:"summaryKey,omitempty"`
	SummaryArgs     map[string]any `json:"summaryArgs,omitempty"`
}

type CompleteResult struct {
	Completed bool         `json:"completed"`
	Report    InitReport   `json:"report"`
	Steps     []StepReport `json:"steps"`
}

type RunRequest struct {
	SetupToken         string         `json:"setupToken"`
	Mode               Mode           `json:"mode"`
	Values             map[string]any `json:"values"`
	OrgCode            string         `json:"orgCode"`
	OrgName            string         `json:"orgName"`
	Username           string         `json:"username"`
	Email              string         `json:"email"`
	DisplayName        string         `json:"displayName"`
	Password           string         `json:"password"`
	CreateServiceToken bool           `json:"createServiceToken"`
	ServiceTokenDays   int            `json:"serviceTokenDays"`
	ServiceTokenRemark string         `json:"serviceTokenRemark"`
}

type ConfigRequest struct {
	SetupToken string         `json:"setupToken"`
	Persist    bool           `json:"persist"`
	Test       bool           `json:"test"`
	Values     map[string]any `json:"values"`
}

type TestRequest struct {
	SetupToken string         `json:"setupToken"`
	Values     map[string]any `json:"values"`
}

type SkipRequest struct {
	SetupToken string `json:"setupToken"`
	Reason     string `json:"reason"`
}

type CompleteRequest struct {
	SetupToken string `json:"setupToken"`
}
