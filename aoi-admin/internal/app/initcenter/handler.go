package initcenter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	appconfig "github.com/rei0721/go-scaffold/internal/config"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/types/result"
)

type Handler struct {
	service *Service
	logger  ports.Logger
	config  HandlerConfig
}

type HandlerConfig struct {
	CookieNamePrefix     string
	CookieDomain         string
	CookiePath           string
	CookieSameSite       string
	CookieSecure         bool
	CSRFEnabled          bool
	CSRFCookieName       string
	ProductHeader        string
	ClientTypeHeader     string
	DefaultProductCode   string
	DefaultClientType    string
	MobileUserAgentHints []string
}

func NewHandler(service *Service, logger ports.Logger, configs ...HandlerConfig) *Handler {
	cfg := HandlerConfig{}
	if len(configs) > 0 {
		cfg = configs[0]
	}
	applyHandlerConfigDefaults(&cfg)
	return &Handler{service: service, logger: logger, config: cfg}
}

func HandlerConfigFromAppConfig(cfg *appconfig.Config) HandlerConfig {
	if cfg == nil {
		return HandlerConfig{}
	}
	return HandlerConfig{
		CookieNamePrefix:     cfg.Auth.Cookie.NamePrefix,
		CookieDomain:         cfg.Auth.Cookie.Domain,
		CookiePath:           cfg.Auth.Cookie.Path,
		CookieSameSite:       cfg.Auth.Cookie.SameSite,
		CookieSecure:         cfg.Auth.Cookie.Secure,
		CSRFEnabled:          cfg.Auth.CSRF.EnabledValue(),
		CSRFCookieName:       cfg.Auth.CSRF.CookieName,
		ProductHeader:        cfg.Auth.Session.ProductHeader,
		ClientTypeHeader:     cfg.Auth.Session.ClientTypeHeader,
		DefaultProductCode:   cfg.Brand.ProductCode,
		DefaultClientType:    cfg.Auth.Session.DefaultClientType,
		MobileUserAgentHints: append([]string(nil), cfg.Auth.Session.MobileUserAgentHints...),
	}
}

func applyHandlerConfigDefaults(cfg *HandlerConfig) {
	if strings.TrimSpace(cfg.CookieNamePrefix) == "" {
		cfg.CookieNamePrefix = "aoi"
	}
	if strings.TrimSpace(cfg.CookiePath) == "" {
		cfg.CookiePath = "/"
	}
	if strings.TrimSpace(cfg.CookieSameSite) == "" {
		cfg.CookieSameSite = "lax"
	}
	if strings.TrimSpace(cfg.CSRFCookieName) == "" {
		cfg.CSRFCookieName = cfg.CookieNamePrefix + "_csrf"
	}
	if strings.TrimSpace(cfg.ProductHeader) == "" {
		cfg.ProductHeader = "X-Aoi-Product-Code"
	}
	if strings.TrimSpace(cfg.ClientTypeHeader) == "" {
		cfg.ClientTypeHeader = "X-Aoi-Client-Type"
	}
	if strings.TrimSpace(cfg.DefaultProductCode) == "" {
		cfg.DefaultProductCode = "platform"
	}
	if strings.TrimSpace(cfg.DefaultClientType) == "" {
		cfg.DefaultClientType = "pc_web"
	}
	if len(cfg.MobileUserAgentHints) == 0 {
		cfg.MobileUserAgentHints = []string{"mobile", "android", "iphone", "ipad"}
	}
}

func (h *Handler) Status(c ports.HTTPContext) {
	status, err := h.service.Status(c.RequestContext())
	if err == nil {
		status = localizeStatus(c, status)
	}
	h.write(c, status, err)
}

func (h *Handler) Schema(c ports.HTTPContext) {
	schema, err := h.service.Schema(c.RequestContext())
	if err == nil {
		schema = localizeSetupSchema(c, schema)
	}
	h.write(c, schema, err)
}

func (h *Handler) CreateRun(c ports.HTTPContext) {
	input, ok := h.bindRunInput(c)
	if !ok {
		return
	}
	run, err := h.service.Run(c.RequestContext(), input)
	if err == nil {
		run = localizeRunResult(c, run)
	}
	h.writeSetup(c, run, err)
}

func (h *Handler) RetryRun(c ports.HTTPContext) {
	input, ok := h.bindRunInput(c)
	if !ok {
		return
	}
	run, err := h.service.Retry(c.RequestContext(), c.Param("id"), input)
	if err == nil {
		run = localizeRunResult(c, run)
	}
	h.writeSetup(c, run, err)
}

func (h *Handler) Logs(c ports.HTTPContext) {
	if err := h.service.AuthorizeSetupRead(c.RequestContext(), setupTokenFromRequest(c)); err != nil {
		h.writeError(c, err)
		return
	}
	logs, err := h.service.Logs(c.RequestContext(), c.Param("id"))
	if err == nil {
		logs = localizeStepReports(c, logs)
	}
	h.write(c, logs, err)
}

func (h *Handler) SaveConfig(c ports.HTTPContext) {
	var req ConfigRequest
	if err := c.BindJSON(&req); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return
	}
	input := h.inputFromSetupToken(c, req.SetupToken)
	saved, err := h.service.SaveConfig(c.RequestContext(), c.Param("stepKey"), input, req.Values, req.Persist, req.Test)
	if err == nil {
		saved.Steps = localizeStepReports(c, saved.Steps)
		if saved.Test != nil {
			test := localizeTestResult(c, *saved.Test)
			saved.Test = &test
		}
	}
	h.write(c, saved, err)
}

func (h *Handler) TestConfig(c ports.HTTPContext) {
	var req TestRequest
	if err := c.BindJSON(&req); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return
	}
	input := h.inputFromSetupToken(c, req.SetupToken)
	test, err := h.service.TestConfig(c.RequestContext(), c.Param("stepKey"), input, req.Values)
	if err == nil {
		test = localizeTestResult(c, test)
	}
	h.write(c, test, err)
}

func (h *Handler) SkipStep(c ports.HTTPContext) {
	var req SkipRequest
	if err := c.BindJSON(&req); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return
	}
	input := h.inputFromSetupToken(c, req.SetupToken)
	run, err := h.service.SkipStep(c.RequestContext(), c.Param("id"), c.Param("stepKey"), req.Reason, input)
	if err == nil {
		run = localizeRunResult(c, run)
	}
	h.write(c, run, err)
}

func (h *Handler) Complete(c ports.HTTPContext) {
	var req CompleteRequest
	if err := c.BindJSON(&req); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return
	}
	completed, err := h.service.Complete(c.RequestContext(), h.inputFromSetupToken(c, req.SetupToken))
	if err == nil {
		completed.Report = localizeInitReport(c, completed.Report)
		completed.Steps = localizeStepReports(c, completed.Steps)
	}
	h.write(c, completed, err)
}

func (h *Handler) bindRunInput(c ports.HTTPContext) (Input, bool) {
	var req RunRequest
	if err := c.BindJSON(&req); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return Input{}, false
	}
	mode := req.Mode
	if mode == "" {
		mode = ModeFirstRun
	}
	applyRunRequestValues(&req)
	return Input{
		Source:             SourceWeb,
		Mode:               mode,
		SetupToken:         fallback(req.SetupToken, setupTokenFromRequest(c)),
		OrgCode:            req.OrgCode,
		OrgName:            req.OrgName,
		AdminUsername:      req.Username,
		AdminEmail:         req.Email,
		AdminDisplayName:   req.DisplayName,
		AdminPassword:      req.Password,
		UserAgent:          c.GetHeader("User-Agent"),
		IPAddress:          c.ClientIP(),
		ProductCode:        h.productCodeFromRequest(c),
		ClientType:         h.clientTypeFromRequest(c),
		IssueLoginTokens:   true,
		CreateServiceToken: req.CreateServiceToken,
		ServiceTokenDays:   req.ServiceTokenDays,
		ServiceTokenRemark: req.ServiceTokenRemark,
	}, true
}

func (h *Handler) inputFromSetupToken(c ports.HTTPContext, setupToken string) Input {
	return Input{
		Source:      SourceWeb,
		Mode:        ModeRepair,
		SetupToken:  fallback(setupToken, setupTokenFromRequest(c)),
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
		ProductCode: h.productCodeFromRequest(c),
		ClientType:  h.clientTypeFromRequest(c),
	}
}

func (h *Handler) productCodeFromRequest(c ports.HTTPContext) string {
	productCode := strings.TrimSpace(c.GetHeader(h.config.ProductHeader))
	if productCode == "" {
		productCode = h.config.DefaultProductCode
	}
	return productCode
}

func (h *Handler) clientTypeFromRequest(c ports.HTTPContext) string {
	clientType := strings.TrimSpace(c.GetHeader(h.config.ClientTypeHeader))
	if clientType != "" {
		return clientType
	}
	ua := strings.ToLower(c.GetHeader("User-Agent"))
	for _, hint := range h.config.MobileUserAgentHints {
		if hint = strings.ToLower(strings.TrimSpace(hint)); hint != "" && strings.Contains(ua, hint) {
			return "mobile_web"
		}
	}
	return h.config.DefaultClientType
}

func (h *Handler) write(c ports.HTTPContext, data any, err error) {
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, h.prepareResponseData(c, data))
}

func (h *Handler) writeSetup(c ports.HTTPContext, data any, err error) {
	if err != nil {
		h.writeError(c, err)
		return
	}
	data = h.prepareResponseData(c, data)
	key := result.MessageKeySuccess
	c.JSON(http.StatusCreated, result.SuccessWithMessage(data, key, localizeAPIMessage(c, key, nil), nil))
}

func (h *Handler) prepareResponseData(c ports.HTTPContext, data any) any {
	if run, ok := data.(RunResult); ok {
		if run.LoginTokensIssued {
			h.setAuthCookies(c, run.RawLoginTokens)
		}
		run.RawLoginTokens = iamservice.TokenPair{}
		return run
	}
	return data
}

func (h *Handler) setAuthCookies(c ports.HTTPContext, pair iamservice.TokenPair) {
	if strings.TrimSpace(pair.AccessToken) == "" || strings.TrimSpace(pair.RefreshToken) == "" {
		return
	}
	h.setCookie(c, h.accessCookieName(), pair.AccessToken, pair.AccessExpiresAt, true)
	h.setCookie(c, h.refreshCookieName(), pair.RefreshToken, pair.RefreshExpiresAt, true)
	if h.config.CSRFEnabled {
		h.setCookie(c, h.config.CSRFCookieName, newSetupCSRFToken(), pair.RefreshExpiresAt, false)
	}
}

func (h *Handler) setCookie(c ports.HTTPContext, name string, value string, expires time.Time, httpOnly bool) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     h.config.CookiePath,
		Domain:   strings.TrimSpace(h.config.CookieDomain),
		Expires:  expires,
		HttpOnly: httpOnly,
		Secure:   h.config.CookieSecure,
		SameSite: setupCookieSameSite(h.config.CookieSameSite),
	}
	if seconds := int(time.Until(expires).Seconds()); seconds > 0 {
		cookie.MaxAge = seconds
	}
	c.SetCookie(cookie)
}

func (h *Handler) accessCookieName() string {
	return strings.TrimSpace(h.config.CookieNamePrefix) + "_access"
}

func (h *Handler) refreshCookieName() string {
	return strings.TrimSpace(h.config.CookieNamePrefix) + "_refresh"
}

func setupCookieSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func newSetupCSRFToken() string {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return base64.RawURLEncoding.EncodeToString(raw[:])
}

func (h *Handler) writeError(c ports.HTTPContext, err error) {
	switch {
	case errors.Is(err, iamservice.ErrInvalidInput):
		result.BadRequest(c, "api.common.invalidRequest")
	case errors.Is(err, iamservice.ErrSetupCompleted), errors.Is(err, ErrSetupUnauthorized):
		result.Forbidden(c, "api.setup.forbidden")
	case errors.Is(err, ErrInitializationRunAbsent):
		result.NotFound(c, "api.setup.runNotFound")
	case errors.Is(err, iamservice.ErrDuplicate):
		result.BadRequest(c, "api.common.duplicate")
	default:
		if h.logger != nil {
			h.logger.Error("setup request failed", "error", err)
		}
		message := strings.TrimSpace(err.Error())
		if message == "" {
			message = "initialization failed"
		}
		result.InternalError(c, "api.setup.failed", map[string]any{"error": message})
	}
}

func localizeSetupSchema(c ports.HTTPContext, schema SetupSchema) SetupSchema {
	for stepIndex := range schema.Steps {
		step := &schema.Steps[stepIndex]
		step.Title = localizeUIMessage(c, step.TitleKey, step.Title, nil)
		step.Description = localizeUIMessage(c, step.DescriptionKey, step.Description, nil)
		for groupIndex := range step.Groups {
			group := &step.Groups[groupIndex]
			group.Title = localizeUIMessage(c, group.TitleKey, group.Title, nil)
			group.Description = localizeUIMessage(c, group.DescriptionKey, group.Description, nil)
			for fieldIndex := range group.Fields {
				localizeField(c, &group.Fields[fieldIndex])
			}
		}
		for fieldIndex := range step.Fields {
			localizeField(c, &step.Fields[fieldIndex])
		}
	}
	return schema
}

func localizeStatus(c ports.HTTPContext, status Status) Status {
	status.Steps = localizeStepReports(c, status.Steps)
	if status.Report != nil {
		report := localizeInitReport(c, *status.Report)
		status.Report = &report
	}
	return status
}

func localizeRunResult(c ports.HTTPContext, run RunResult) RunResult {
	run.Report = localizeInitReport(c, run.Report)
	run.Steps = localizeStepReports(c, run.Steps)
	return run
}

func localizeInitReport(c ports.HTTPContext, report InitReport) InitReport {
	if strings.TrimSpace(report.SummaryKey) != "" {
		report.Summary = localizeUIMessage(c, report.SummaryKey, report.Summary, report.SummaryArgs)
		return report
	}
	switch {
	case report.RestartRequired:
		report.SummaryKey = "ui.setup.report.restartRequired"
		report.Summary = localizeUIMessage(c, report.SummaryKey, report.Summary, nil)
	case report.Failed > 0:
		report.SummaryKey = "ui.setup.report.failed"
		report.SummaryArgs = map[string]any{"count": report.Failed}
		report.Summary = localizeUIMessage(c, report.SummaryKey, report.Summary, report.SummaryArgs)
	case report.Risk > 0:
		report.SummaryKey = "ui.setup.report.risk"
		report.SummaryArgs = map[string]any{"count": report.Risk}
		report.Summary = localizeUIMessage(c, report.SummaryKey, report.Summary, report.SummaryArgs)
	default:
		report.SummaryKey = "ui.setup.report.passed"
		report.Summary = localizeUIMessage(c, report.SummaryKey, report.Summary, nil)
	}
	return report
}

func localizeStepReports(c ports.HTTPContext, steps []StepReport) []StepReport {
	if steps == nil {
		return []StepReport{}
	}
	out := append([]StepReport(nil), steps...)
	for index := range out {
		localizeStepReport(c, &out[index])
	}
	return out
}

func localizeStepReport(c ports.HTTPContext, step *StepReport) {
	if step == nil {
		return
	}
	if step.TitleKey == "" {
		step.TitleKey = "ui.setup.steps." + step.Key + ".title"
	}
	if step.GoalKey == "" {
		step.GoalKey = "ui.setup.steps." + step.Key + ".description"
	}
	step.Title = localizeUIMessage(c, step.TitleKey, step.Title, nil)
	step.Goal = localizeUIMessage(c, step.GoalKey, step.Goal, nil)
	step.Prerequisites = localizeUIList(c, step.PrerequisiteKeys, step.Prerequisites)
	step.UserInputs = localizeUIList(c, step.UserInputKeys, step.UserInputs)
	step.AutomaticActions = localizeUIList(c, step.AutomaticActionKeys, step.AutomaticActions)
	step.CompletionMark = localizeUIMessage(c, step.CompletionMarkKey, step.CompletionMark, nil)
	step.Recovery = localizeUIMessage(c, step.RecoveryKey, step.Recovery, nil)
	ensureStepRuntimeMessageKeys(step)
	step.OutputSummary = localizeOptionalUIMessage(c, step.OutputSummaryKey, step.OutputSummary, step.OutputSummaryArgs)
	step.ErrorMessage = localizeOptionalUIMessage(c, step.ErrorMessageKey, step.ErrorMessage, step.ErrorMessageArgs)
	step.RepairHint = localizeOptionalUIMessage(c, step.RepairHintKey, step.RepairHint, step.RepairHintArgs)
	step.SkippedReason = localizeOptionalUIMessage(c, step.SkippedReasonKey, step.SkippedReason, step.SkippedReasonArgs)
	step.TestSummary = localizeOptionalUIMessage(c, step.TestSummaryKey, step.TestSummary, step.TestSummaryArgs)
	step.TestError = localizeOptionalUIMessage(c, step.TestErrorKey, step.TestError, step.TestErrorArgs)
	if step.Schema.Key != "" {
		localized := localizeSetupSchema(c, SetupSchema{Steps: []StepSchema{step.Schema}})
		if len(localized.Steps) > 0 {
			step.Schema = localized.Steps[0]
		}
	}
}

func ensureStepRuntimeMessageKeys(step *StepReport) {
	if step == nil {
		return
	}
	stepName := strings.TrimSpace(step.Title)
	if stepName == "" {
		stepName = step.Key
	}
	if strings.TrimSpace(step.OutputSummary) != "" && strings.TrimSpace(step.OutputSummaryKey) == "" {
		step.OutputSummaryKey = stepStatusSummaryKey(step.Status)
		if len(step.OutputSummaryArgs) == 0 {
			step.OutputSummaryArgs = map[string]any{"step": stepName}
		}
	}
	if strings.TrimSpace(step.TestSummary) != "" && strings.TrimSpace(step.TestSummaryKey) == "" {
		step.TestSummaryKey = testStatusSummaryKey(step.TestStatus)
		if len(step.TestSummaryArgs) == 0 {
			step.TestSummaryArgs = map[string]any{"step": stepName}
		}
	}
}

func stepStatusSummaryKey(status StepStatus) string {
	switch status {
	case StepStatusSucceeded:
		return "ui.setup.stepStatus.succeeded"
	case StepStatusFailed:
		return "ui.setup.stepStatus.failed"
	case StepStatusSkipped:
		return "ui.setup.stepStatus.skipped"
	case StepStatusRunning:
		return "ui.setup.stepStatus.running"
	default:
		return "ui.setup.stepStatus.pending"
	}
}

func testStatusSummaryKey(status string) string {
	switch strings.TrimSpace(status) {
	case "succeeded":
		return "ui.setup.tests.succeeded"
	case "failed":
		return "ui.setup.tests.failed"
	case "skipped":
		return "ui.setup.tests.skipped"
	default:
		return "ui.setup.tests.skipped"
	}
}

func localizeTestResult(c ports.HTTPContext, test TestResult) TestResult {
	stepTitle := localizeUIMessage(c, "ui.setup.steps."+test.StepKey+".title", test.StepKey, nil)
	switch test.Status {
	case "succeeded":
		if strings.TrimSpace(test.SummaryKey) == "" {
			test.SummaryKey = "ui.setup.tests.succeeded"
		}
	case "failed":
		if strings.TrimSpace(test.SummaryKey) == "" {
			test.SummaryKey = "ui.setup.tests.failed"
		}
	case "skipped":
		if strings.TrimSpace(test.SummaryKey) == "" {
			test.SummaryKey = "ui.setup.tests.skipped"
		}
	}
	if strings.HasPrefix(test.SummaryKey, "ui.setup.tests.") && len(test.SummaryArgs) == 0 {
		test.SummaryArgs = map[string]any{"step": stepTitle}
	}
	if strings.TrimSpace(test.RepairHint) != "" && strings.TrimSpace(test.RepairHintKey) == "" {
		test.RepairHintKey = "ui.setup.testRepair." + sanitizeKeyPart(test.StepKey)
	}
	test.Summary = localizeOptionalUIMessage(c, test.SummaryKey, test.Summary, test.SummaryArgs)
	test.Error = localizeOptionalUIMessage(c, test.ErrorKey, test.Error, test.ErrorArgs)
	test.RepairHint = localizeOptionalUIMessage(c, test.RepairHintKey, test.RepairHint, test.RepairHintArgs)
	return test
}

func localizeField(c ports.HTTPContext, field *FieldSchema) {
	field.Label = localizeUIMessage(c, field.LabelKey, field.Label, nil)
	field.Help = localizeUIMessage(c, field.HelpKey, field.Help, nil)
	for optionIndex := range field.Options {
		option := &field.Options[optionIndex]
		option.Label = localizeUIMessage(c, option.LabelKey, option.Label, nil)
	}
}

func localizeUIMessage(c ports.HTTPContext, fullKey string, fallback string, args map[string]any) string {
	return localizeMessage(c, fullKey, fallback, args)
}

func localizeOptionalUIMessage(c ports.HTTPContext, fullKey string, fallback string, args map[string]any) string {
	if strings.TrimSpace(fullKey) == "" {
		return fallback
	}
	return localizeUIMessage(c, fullKey, fallback, args)
}

func localizeUIList(c ports.HTTPContext, keys []string, fallbacks []string) []string {
	if fallbacks == nil {
		fallbacks = []string{}
	}
	out := make([]string, len(fallbacks))
	for index, fallback := range fallbacks {
		key := ""
		if index < len(keys) {
			key = keys[index]
		}
		out[index] = localizeOptionalUIMessage(c, key, fallback, nil)
	}
	return out
}

func localizeAPIMessage(c ports.HTTPContext, fullKey string, args map[string]any) string {
	return localizeMessage(c, fullKey, fullKey, args)
}

func localizeMessage(c ports.HTTPContext, fullKey string, fallback string, args map[string]any) string {
	namespace, key := splitFullMessageKey(fullKey)
	if value, ok := c.Get("i18n"); ok {
		if localizer, ok := value.(ports.I18n); ok {
			locale := localizer.DefaultLocale()
			if raw, ok := c.Get("locale"); ok {
				if text, ok := raw.(string); ok && strings.TrimSpace(text) != "" {
					locale = text
				}
			}
			resolved := localizer.Localize(locale, namespace, key, args)
			if resolved != key && resolved != fullKey {
				return resolved
			}
		}
	}
	return fallback
}

func splitFullMessageKey(fullKey string) (string, string) {
	fullKey = strings.TrimSpace(fullKey)
	parts := strings.SplitN(fullKey, ".", 2)
	if len(parts) != 2 {
		return "ui", fullKey
	}
	return parts[0], parts[1]
}

func setupTokenFromRequest(c ports.HTTPContext) string {
	if token := strings.TrimSpace(c.GetHeader("X-Setup-Token")); token != "" {
		return token
	}
	req := c.Request()
	if req == nil || req.URL == nil {
		return ""
	}
	return strings.TrimSpace(req.URL.Query().Get("setupToken"))
}

type IAMSetupService interface {
	SetupStatus(context.Context) (iamservice.SetupStatus, error)
	InitialAdminSetup(context.Context, iamservice.InitialAdminSetupInput) (iamservice.TokenPair, error)
}

type iamSetupAdapter struct {
	center *Service
}

func (s iamSetupAdapter) SetupStatus(ctx context.Context) (iamservice.SetupStatus, error) {
	status, err := s.center.Status(ctx)
	if err != nil {
		return iamservice.SetupStatus{}, err
	}
	return iamservice.SetupStatus{Required: status.Required, PasswordPolicy: status.PasswordPolicy}, nil
}

func applyRunRequestValues(r *RunRequest) {
	if r == nil || len(r.Values) == 0 {
		return
	}
	r.OrgCode = fallback(r.OrgCode, mapString(r.Values, "orgCode"))
	r.OrgName = fallback(r.OrgName, mapString(r.Values, "orgName"))
	r.Username = fallback(r.Username, mapString(r.Values, "username"))
	r.Email = fallback(r.Email, mapString(r.Values, "email"))
	r.DisplayName = fallback(r.DisplayName, mapString(r.Values, "displayName"))
	r.Password = fallback(r.Password, mapString(r.Values, "password"))
	if value, ok := r.Values["createServiceToken"]; ok && !r.CreateServiceToken {
		r.CreateServiceToken = boolValue(value)
	}
	if value, ok := r.Values["serviceTokenDays"]; ok && r.ServiceTokenDays == 0 {
		r.ServiceTokenDays = intValue(value)
	}
	r.ServiceTokenRemark = fallback(r.ServiceTokenRemark, mapString(r.Values, "serviceTokenRemark"))
}

func mapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key]; ok {
		return stringValue(value)
	}
	return ""
}

func (s iamSetupAdapter) InitialAdminSetup(ctx context.Context, input iamservice.InitialAdminSetupInput) (iamservice.TokenPair, error) {
	run, err := s.center.Run(ctx, Input{
		Source:           SourceWeb,
		Mode:             ModeFirstRun,
		OrgCode:          input.OrgCode,
		OrgName:          input.OrgName,
		AdminUsername:    input.Username,
		AdminEmail:       input.Email,
		AdminDisplayName: input.DisplayName,
		AdminPassword:    input.Password,
		ProductCode:      input.ProductCode,
		ClientType:       input.ClientType,
		UserAgent:        input.UserAgent,
		IPAddress:        input.IPAddress,
		IssueLoginTokens: true,
	})
	if err != nil {
		return iamservice.TokenPair{}, err
	}
	return run.RawLoginTokens, nil
}
