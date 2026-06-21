package adapters

import (
	"context"
	"fmt"
	"strings"

	systemmodel "github.com/rei0721/go-scaffold/internal/modules/system/model"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/pkg/mail"
)

type TrafficAlertSink struct {
	logger ports.Logger
	sender mail.Sender
}

func NewTrafficAlertSink(logger ports.Logger, sender mail.Sender) TrafficAlertSink {
	return TrafficAlertSink{logger: logger, sender: sender}
}

func (s TrafficAlertSink) NotifyTrafficHijack(ctx context.Context, target systemmodel.TrafficProbeTarget, event systemmodel.TrafficHijackEvent, result systemmodel.TrafficProbeResult, transition string) string {
	channels := alertChannels(target.AlertChannels)
	status := make([]string, 0, len(channels))
	if channels["event"] {
		status = append(status, "event")
	}
	if channels["debug"] {
		status = append(status, "debug")
		if s.logger != nil {
			s.logger.Warn(
				"traffic hijack event changed",
				"transition", transition,
				"target_id", target.ID,
				"target_name", target.Name,
				"target_url", target.URL,
				"reason", event.Reason,
				"severity", event.Severity,
				"state", event.State,
				"occurrences", event.Occurrences,
			)
		}
	}
	if channels["email"] {
		emailStatus, err := s.sendEmail(ctx, target, event, result, transition)
		status = append(status, emailStatus)
		if err != nil {
			return strings.Join(status, ",")
		}
	}
	if len(status) == 0 {
		status = append(status, "none")
	}
	return strings.Join(status, ",")
}

func (s TrafficAlertSink) sendEmail(ctx context.Context, target systemmodel.TrafficProbeTarget, event systemmodel.TrafficHijackEvent, result systemmodel.TrafficProbeResult, transition string) (string, error) {
	recipients := splitCSV(target.EmailRecipients)
	if len(recipients) == 0 {
		return "email_skipped:no_recipients", nil
	}
	if s.sender == nil {
		return "email_skipped:no_sender", nil
	}
	subject := fmt.Sprintf("[Aoi Admin] Traffic probe %s: %s", transition, target.Name)
	body := fmt.Sprintf(
		"Traffic hijack monitor event %s\n\nTarget: %s\nURL: %s\nReason: %s\nSeverity: %s\nState: %s\nOccurrences: %d\nStatus code: %d\nFinal URL: %s\nStage: %s\nEvidence: %s\n",
		transition,
		target.Name,
		target.URL,
		event.Reason,
		event.Severity,
		event.State,
		event.Occurrences,
		result.StatusCode,
		result.FinalURL,
		result.Stage,
		event.EvidenceJSON,
	)
	if err := s.sender.Send(ctx, mail.Message{
		To:       recipients,
		Subject:  subject,
		TextBody: body,
	}); err != nil {
		if s.logger != nil {
			s.logger.Warn("traffic hijack email alert failed", "target_id", target.ID, "event_id", event.ID, "error", err)
		}
		return "email_failed", err
	}
	return "email_sent", nil
}

func alertChannels(value string) map[string]bool {
	out := map[string]bool{}
	for _, item := range splitCSV(value) {
		switch strings.ToLower(item) {
		case "event", "debug", "email":
			out[strings.ToLower(item)] = true
		}
	}
	if len(out) == 0 {
		out["event"] = true
	}
	return out
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
}
