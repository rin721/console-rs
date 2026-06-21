package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
)

const (
	defaultTrafficProbeIntervalSeconds = 30
	defaultTrafficProbeTimeoutSeconds  = 5
	defaultTrafficProbeStatusCodes     = "200-399"
	defaultTrafficProbeResultKeep      = 500
)

func (s *service) GetTrafficHijackOverview(ctx context.Context) (model.TrafficHijackOverview, error) {
	targets, err := s.ListTrafficProbeTargets(ctx)
	if err != nil {
		if isStorageUnavailable(err) {
			return model.TrafficHijackOverview{}, nil
		}
		return model.TrafficHijackOverview{}, err
	}
	resultPage, err := s.ListTrafficProbeResults(ctx, TrafficProbeResultFilter{Limit: 12})
	if err != nil {
		return model.TrafficHijackOverview{}, err
	}
	eventPage, err := s.ListTrafficHijackEvents(ctx, TrafficHijackEventFilter{State: model.TrafficHijackEventStateOpen, Page: 1, PageSize: 12})
	if err != nil {
		return model.TrafficHijackOverview{}, err
	}
	overview := model.TrafficHijackOverview{
		TotalTargets:   len(targets),
		RecentResults:  resultPage.Items,
		RecentEvents:   eventPage.Items,
		OpenEvents:     int(eventPage.Total),
		EnabledTargets: 0,
	}
	for _, target := range targets {
		if target.Enabled {
			overview.EnabledTargets++
		}
		switch normalizeTrafficStatus(target.LastStatus, target.LastSeverity) {
		case model.TrafficProbeStatusHealthy:
			overview.HealthyTargets++
		case model.TrafficProbeStatusWarning:
			overview.WarningTargets++
		case model.TrafficProbeStatusCritical:
			overview.CriticalTargets++
		}
		if target.LastProbedAt != nil && (overview.LastProbeAt == nil || target.LastProbedAt.After(*overview.LastProbeAt)) {
			probedAt := target.LastProbedAt.UTC()
			overview.LastProbeAt = &probedAt
		}
	}
	return overview, nil
}

func (s *service) ListTrafficProbeTargets(ctx context.Context) ([]model.TrafficProbeTarget, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	targets, err := s.repo.ListTrafficProbeTargets(ctx)
	if err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].Enabled != targets[j].Enabled {
			return targets[i].Enabled
		}
		return targets[i].CreatedAt.After(targets[j].CreatedAt)
	})
	return targets, nil
}

func (s *service) CreateTrafficProbeTarget(ctx context.Context, input CreateTrafficProbeTargetInput) (*model.TrafficProbeTarget, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	now := s.now()
	target, err := s.newTrafficProbeTarget(input, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateTrafficProbeTarget(ctx, target); err != nil {
		if isStorageUnavailable(err) {
			return nil, ErrStorageUnavailable
		}
		return nil, err
	}
	s.publishTrafficEvent("target", target)
	return target, nil
}

func (s *service) UpdateTrafficProbeTarget(ctx context.Context, id int64, input UpdateTrafficProbeTargetInput) (*model.TrafficProbeTarget, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	target, err := s.repo.FindTrafficProbeTargetByID(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	if err := applyTrafficProbeTargetUpdate(target, input, s.now()); err != nil {
		return nil, err
	}
	if err := s.repo.SaveTrafficProbeTarget(ctx, target); err != nil {
		return nil, mapRepositoryError(err)
	}
	s.publishTrafficEvent("target", target)
	return target, nil
}

func (s *service) DeleteTrafficProbeTarget(ctx context.Context, id int64) error {
	if s.repo == nil {
		return ErrStorageUnavailable
	}
	if id <= 0 {
		return ErrInvalidInput
	}
	if err := s.repo.DeleteTrafficProbeTarget(ctx, id, s.now()); err != nil {
		return mapRepositoryError(err)
	}
	s.publishTrafficEvent("targetDeleted", map[string]string{"id": strconv.FormatInt(id, 10)})
	return nil
}

func (s *service) ListTrafficProbeResults(ctx context.Context, filter TrafficProbeResultFilter) (model.TrafficProbeResultPage, error) {
	result := model.TrafficProbeResultPage{Limit: normalizeTrafficResultLimit(filter.Limit), StorageStatus: "memory"}
	if s.repo == nil {
		return result, nil
	}
	filter.Limit = result.Limit + 1
	items, err := s.repo.ListTrafficProbeResults(ctx, filter)
	if err != nil {
		if isStorageUnavailable(err) {
			result.StorageStatus = "unavailable"
			return result, nil
		}
		return result, err
	}
	result.StorageStatus = "persisted"
	if len(items) > result.Limit {
		result.NextCursor = items[result.Limit-1].ID
		items = items[:result.Limit]
	}
	result.Items = items
	return result, nil
}

func (s *service) ListTrafficHijackEvents(ctx context.Context, filter TrafficHijackEventFilter) (model.TrafficHijackEventPage, error) {
	page := normalizePage(filter.Page)
	pageSize := normalizePageSize(filter.PageSize)
	result := model.TrafficHijackEventPage{Page: page, PageSize: pageSize, StorageStatus: "memory"}
	if s.repo == nil {
		return result, nil
	}
	filter.Page, filter.PageSize = page, pageSize
	items, total, err := s.repo.ListTrafficHijackEvents(ctx, filter)
	if err != nil {
		if isStorageUnavailable(err) {
			result.StorageStatus = "unavailable"
			return result, nil
		}
		return result, err
	}
	result.Items = items
	result.Total = total
	result.StorageStatus = "persisted"
	return result, nil
}

func (s *service) ResolveTrafficHijackEvent(ctx context.Context, id int64) (*model.TrafficHijackEvent, error) {
	if s.repo == nil {
		return nil, ErrStorageUnavailable
	}
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	event, err := s.repo.FindTrafficHijackEvent(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	now := s.now()
	event.State = model.TrafficHijackEventStateResolved
	event.ResolvedAt = &now
	event.UpdatedAt = now
	if err := s.repo.SaveTrafficHijackEvent(ctx, event); err != nil {
		return nil, mapRepositoryError(err)
	}
	s.publishTrafficEvent("eventResolved", event)
	return event, nil
}

func (s *service) RunDueTrafficProbes(ctx context.Context) (int, error) {
	targets, err := s.ListTrafficProbeTargets(ctx)
	if err != nil {
		return 0, err
	}
	now := s.now()
	count := 0
	for _, target := range targets {
		if !target.Enabled {
			continue
		}
		if target.NextProbeAt != nil && target.NextProbeAt.After(now) {
			continue
		}
		if _, err := s.RunTrafficProbe(ctx, target.ID); err != nil && !isStorageUnavailable(err) {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *service) RunTrafficProbe(ctx context.Context, id int64) (model.TrafficProbeResult, error) {
	if s.repo == nil {
		return model.TrafficProbeResult{}, ErrStorageUnavailable
	}
	if id <= 0 {
		return model.TrafficProbeResult{}, ErrInvalidInput
	}
	target, err := s.repo.FindTrafficProbeTargetByID(ctx, id)
	if err != nil {
		return model.TrafficProbeResult{}, mapRepositoryError(err)
	}
	result := s.trafficRunner.Probe(ctx, *target)
	now := s.now()
	applyTrafficProbeResultDefaults(&result, *target, now, s.ids.NextID())
	if err := s.repo.CreateTrafficProbeResult(ctx, &result); err != nil {
		return model.TrafficProbeResult{}, mapRepositoryError(err)
	}
	_ = s.repo.DeleteOldTrafficProbeResults(ctx, target.ID, defaultTrafficProbeResultKeep)
	updateTrafficProbeTargetAfterResult(target, result, now)
	if err := s.repo.SaveTrafficProbeTarget(ctx, target); err != nil {
		return model.TrafficProbeResult{}, mapRepositoryError(err)
	}
	if trafficProbeResultIsHealthy(result) {
		if err := s.resolveOpenTrafficHijackEvents(ctx, *target, result); err != nil {
			return result, err
		}
	} else if err := s.recordTrafficHijackEvent(ctx, *target, result); err != nil {
		return result, err
	}
	s.publishTrafficEvent("result", result)
	s.publishTrafficEvent("target", target)
	return result, nil
}

func (s *service) SubscribeTrafficHijack(ctx context.Context) (<-chan model.TrafficHijackStreamEvent, func()) {
	ch := make(chan model.TrafficHijackStreamEvent, 32)
	s.trafficMu.Lock()
	s.trafficNextSub++
	id := s.trafficNextSub
	s.trafficSubs[id] = ch
	s.trafficMu.Unlock()
	cancel := func() {
		s.trafficMu.Lock()
		if current, ok := s.trafficSubs[id]; ok {
			delete(s.trafficSubs, id)
			close(current)
		}
		s.trafficMu.Unlock()
	}
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ch, cancel
}

func (s *service) newTrafficProbeTarget(input CreateTrafficProbeTargetInput, now time.Time) (*model.TrafficProbeTarget, error) {
	target := &model.TrafficProbeTarget{
		ID:                     s.ids.NextID(),
		Name:                   input.Name,
		URL:                    input.URL,
		Method:                 input.Method,
		Enabled:                input.Enabled,
		IntervalSeconds:        input.IntervalSeconds,
		TimeoutSeconds:         input.TimeoutSeconds,
		ExpectedStatusCodes:    input.ExpectedStatusCodes,
		ExpectedFinalHost:      input.ExpectedFinalHost,
		ExpectedContentKeyword: input.ExpectedContentKeyword,
		ExpectedIPCIDRs:        strings.Join(input.ExpectedIPCIDRs, ","),
		ExpectedTLSFingerprint: input.ExpectedTLSFingerprint,
		AllowPrivateNetwork:    input.AllowPrivateNetwork,
		AlertChannels:          strings.Join(input.AlertChannels, ","),
		EmailRecipients:        strings.Join(input.EmailRecipients, ","),
		LastStatus:             model.TrafficProbeStatusPending,
		LastSeverity:           model.TrafficProbeSeverityOK,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	target.NextProbeAt = &now
	if err := normalizeTrafficProbeTarget(target); err != nil {
		return nil, err
	}
	return target, nil
}

func applyTrafficProbeTargetUpdate(target *model.TrafficProbeTarget, input UpdateTrafficProbeTargetInput, now time.Time) error {
	if input.Name != nil {
		target.Name = *input.Name
	}
	if input.URL != nil {
		target.URL = *input.URL
	}
	if input.Method != nil {
		target.Method = *input.Method
	}
	if input.Enabled != nil {
		target.Enabled = *input.Enabled
		if *input.Enabled && target.NextProbeAt == nil {
			next := now
			target.NextProbeAt = &next
		}
	}
	if input.IntervalSeconds != nil {
		target.IntervalSeconds = *input.IntervalSeconds
	}
	if input.TimeoutSeconds != nil {
		target.TimeoutSeconds = *input.TimeoutSeconds
	}
	if input.ExpectedStatusCodes != nil {
		target.ExpectedStatusCodes = *input.ExpectedStatusCodes
	}
	if input.ExpectedFinalHost != nil {
		target.ExpectedFinalHost = *input.ExpectedFinalHost
	}
	if input.ExpectedContentKeyword != nil {
		target.ExpectedContentKeyword = *input.ExpectedContentKeyword
	}
	if input.ExpectedIPCIDRs != nil {
		target.ExpectedIPCIDRs = strings.Join(*input.ExpectedIPCIDRs, ",")
	}
	if input.ExpectedTLSFingerprint != nil {
		target.ExpectedTLSFingerprint = *input.ExpectedTLSFingerprint
	}
	if input.AllowPrivateNetwork != nil {
		target.AllowPrivateNetwork = *input.AllowPrivateNetwork
	}
	if input.AlertChannels != nil {
		target.AlertChannels = strings.Join(*input.AlertChannels, ",")
	}
	if input.EmailRecipients != nil {
		target.EmailRecipients = strings.Join(*input.EmailRecipients, ",")
	}
	target.UpdatedAt = now
	return normalizeTrafficProbeTarget(target)
}

func normalizeTrafficProbeTarget(target *model.TrafficProbeTarget) error {
	target.Name = strings.TrimSpace(target.Name)
	target.URL = strings.TrimSpace(target.URL)
	if target.Name == "" || target.URL == "" {
		return ErrInvalidInput
	}
	parsed, err := url.Parse(target.URL)
	if err != nil || parsed.User != nil || parsed.Host == "" {
		return ErrInvalidInput
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return ErrInvalidInput
	}
	target.Method = strings.ToUpper(strings.TrimSpace(target.Method))
	if target.Method == "" {
		target.Method = model.TrafficProbeMethodGET
	}
	if target.Method != model.TrafficProbeMethodGET && target.Method != model.TrafficProbeMethodHEAD {
		return ErrInvalidInput
	}
	if target.IntervalSeconds <= 0 {
		target.IntervalSeconds = defaultTrafficProbeIntervalSeconds
	}
	if target.IntervalSeconds < 10 {
		target.IntervalSeconds = 10
	}
	if target.TimeoutSeconds <= 0 {
		target.TimeoutSeconds = defaultTrafficProbeTimeoutSeconds
	}
	if target.TimeoutSeconds > 60 {
		target.TimeoutSeconds = 60
	}
	target.ExpectedStatusCodes = strings.TrimSpace(target.ExpectedStatusCodes)
	if target.ExpectedStatusCodes == "" {
		target.ExpectedStatusCodes = defaultTrafficProbeStatusCodes
	}
	target.ExpectedFinalHost = strings.TrimSpace(strings.ToLower(target.ExpectedFinalHost))
	target.ExpectedContentKeyword = strings.TrimSpace(target.ExpectedContentKeyword)
	target.ExpectedIPCIDRs = joinCleanCSV(splitCSV(target.ExpectedIPCIDRs))
	target.ExpectedTLSFingerprint = normalizeFingerprint(target.ExpectedTLSFingerprint)
	target.AlertChannels = normalizeAlertChannels(target.AlertChannels)
	target.EmailRecipients = joinCleanCSV(splitCSV(target.EmailRecipients))
	if target.LastStatus == "" {
		target.LastStatus = model.TrafficProbeStatusPending
	}
	if target.LastSeverity == "" {
		target.LastSeverity = model.TrafficProbeSeverityOK
	}
	return nil
}

func applyTrafficProbeResultDefaults(result *model.TrafficProbeResult, target model.TrafficProbeTarget, now time.Time, id int64) {
	result.ID = id
	result.TargetID = target.ID
	result.TargetName = target.Name
	result.URL = target.URL
	result.Method = target.Method
	result.CreatedAt = now
	if result.FinalURL == "" {
		result.FinalURL = target.URL
	}
	if result.Status == "" {
		result.Status = normalizeTrafficStatus("", result.Severity)
	}
	if result.Severity == "" {
		result.Severity = model.TrafficProbeSeverityOK
	}
	if result.Reason == "" {
		result.Reason = result.Status
	}
	if result.EvidenceJSON == "" {
		result.EvidenceJSON = "{}"
	}
}

func updateTrafficProbeTargetAfterResult(target *model.TrafficProbeTarget, result model.TrafficProbeResult, now time.Time) {
	target.LastStatus = normalizeTrafficStatus(result.Status, result.Severity)
	target.LastSeverity = result.Severity
	target.LastReason = result.Reason
	target.LastProbedAt = &now
	next := now.Add(time.Duration(target.IntervalSeconds) * time.Second)
	target.NextProbeAt = &next
	target.UpdatedAt = now
}

func (s *service) recordTrafficHijackEvent(ctx context.Context, target model.TrafficProbeTarget, probeResult model.TrafficProbeResult) error {
	reason := strings.TrimSpace(probeResult.Reason)
	if reason == "" {
		reason = "probe anomaly"
	}
	hash := trafficEvidenceHash(target.ID, reason, probeResult.Stage, probeResult.EvidenceJSON)
	now := s.now()
	event, err := s.repo.FindOpenTrafficHijackEvent(ctx, target.ID, reason, hash)
	if err != nil && !errorsIsNotFound(err) {
		return mapRepositoryError(err)
	}
	transition := "updated"
	if event == nil {
		event = &model.TrafficHijackEvent{
			ID:           s.ids.NextID(),
			TargetID:     target.ID,
			TargetName:   target.Name,
			Reason:       reason,
			Severity:     probeResult.Severity,
			State:        model.TrafficHijackEventStateOpen,
			EvidenceHash: hash,
			EvidenceJSON: probeResult.EvidenceJSON,
			FirstSeenAt:  now,
			LastSeenAt:   now,
			Occurrences:  1,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		event.NotificationStatus = s.notifyTrafficHijack(ctx, target, *event, probeResult, "opened")
		if err := s.repo.CreateTrafficHijackEvent(ctx, event); err != nil {
			return mapRepositoryError(err)
		}
		transition = "opened"
	} else {
		event.TargetName = target.Name
		event.Severity = probeResult.Severity
		event.EvidenceJSON = probeResult.EvidenceJSON
		event.LastSeenAt = now
		event.Occurrences++
		event.UpdatedAt = now
		if err := s.repo.SaveTrafficHijackEvent(ctx, event); err != nil {
			return mapRepositoryError(err)
		}
	}
	s.publishTrafficEvent("event"+strings.Title(transition), event)
	return nil
}

func (s *service) resolveOpenTrafficHijackEvents(ctx context.Context, target model.TrafficProbeTarget, probeResult model.TrafficProbeResult) error {
	page, err := s.ListTrafficHijackEvents(ctx, TrafficHijackEventFilter{TargetID: target.ID, State: model.TrafficHijackEventStateOpen, Page: 1, PageSize: 100})
	if err != nil {
		return err
	}
	now := s.now()
	for i := range page.Items {
		event := page.Items[i]
		event.State = model.TrafficHijackEventStateResolved
		event.ResolvedAt = &now
		event.UpdatedAt = now
		event.NotificationStatus = s.notifyTrafficHijack(ctx, target, event, probeResult, "resolved")
		if err := s.repo.SaveTrafficHijackEvent(ctx, &event); err != nil {
			return mapRepositoryError(err)
		}
		s.publishTrafficEvent("eventResolved", event)
	}
	return nil
}

func (s *service) notifyTrafficHijack(ctx context.Context, target model.TrafficProbeTarget, event model.TrafficHijackEvent, result model.TrafficProbeResult, transition string) string {
	if s.trafficAlert == nil {
		return "skipped"
	}
	return s.trafficAlert.NotifyTrafficHijack(ctx, target, event, result, transition)
}

func (s *service) publishTrafficEvent(kind string, payload any) {
	event := model.TrafficHijackStreamEvent{Type: kind, Payload: payload}
	s.trafficMu.RLock()
	defer s.trafficMu.RUnlock()
	for _, sub := range s.trafficSubs {
		select {
		case sub <- event:
		default:
		}
	}
}

func trafficProbeResultIsHealthy(result model.TrafficProbeResult) bool {
	return normalizeTrafficStatus(result.Status, result.Severity) == model.TrafficProbeStatusHealthy
}

func normalizeTrafficStatus(status string, severity string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case model.TrafficProbeStatusHealthy, model.TrafficProbeStatusWarning, model.TrafficProbeStatusCritical, model.TrafficProbeStatusPending:
		return strings.ToLower(strings.TrimSpace(status))
	}
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case model.TrafficProbeSeverityOK:
		return model.TrafficProbeStatusHealthy
	case model.TrafficProbeSeverityLow, model.TrafficProbeSeverityMedium:
		return model.TrafficProbeStatusWarning
	case model.TrafficProbeSeverityHigh, model.TrafficProbeSeverityCritical:
		return model.TrafficProbeStatusCritical
	default:
		return model.TrafficProbeStatusPending
	}
}

func trafficEvidenceHash(targetID int64, reason string, stage string, evidence string) string {
	sum := sha256.Sum256([]byte(strconv.FormatInt(targetID, 10) + "|" + reason + "|" + stage + "|" + evidence))
	return hex.EncodeToString(sum[:])
}

func normalizeTrafficResultLimit(value int) int {
	if value <= 0 {
		return 50
	}
	if value > 200 {
		return 200
	}
	return value
}

func normalizeFingerprint(value string) string {
	replacer := strings.NewReplacer(":", "", " ", "", "-", "")
	return strings.ToLower(strings.TrimSpace(replacer.Replace(value)))
}

func normalizeAlertChannels(value string) string {
	channels := splitCSV(value)
	if len(channels) == 0 {
		return model.TrafficAlertChannelEvent
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(channels))
	for _, channel := range channels {
		channel = strings.ToLower(strings.TrimSpace(channel))
		switch channel {
		case model.TrafficAlertChannelEvent, model.TrafficAlertChannelDebug, model.TrafficAlertChannelEmail:
		default:
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		out = append(out, channel)
	}
	if len(out) == 0 {
		return model.TrafficAlertChannelEvent
	}
	return strings.Join(out, ",")
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func joinCleanCSV(values []string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return strings.Join(out, ",")
}

func evidenceJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	if errorsIsNotFound(err) {
		return ErrNotFound
	}
	if isStorageUnavailable(err) {
		return ErrStorageUnavailable
	}
	return err
}

func errorsIsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
