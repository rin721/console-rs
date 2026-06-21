package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/rei0721/go-scaffold/internal/modules/iam/service"
)

type MailMessage struct {
	To       []string
	Subject  string
	TextBody string
}

type MailSender interface {
	Send(context.Context, MailMessage) error
}

type SMTPNotifierConfig struct {
	Sender       MailSender
	Localize     func(key string, data map[string]any) string
	TemplateData map[string]any
}

type SMTPNotifier struct {
	cfg SMTPNotifierConfig
}

func NewSMTPNotifier(cfg SMTPNotifierConfig) (*SMTPNotifier, error) {
	if cfg.Sender == nil || cfg.Localize == nil {
		return nil, service.ErrInvalidInput
	}
	return &SMTPNotifier{cfg: cfg}, nil
}

func (n *SMTPNotifier) SendInvitation(ctx context.Context, notice service.InvitationNotice) error {
	data := n.templateData(notice.URL)
	subject, err := n.localize("email.invitation.subject", data)
	if err != nil {
		return err
	}
	body, err := n.localize("email.invitation.body", data)
	if err != nil {
		return err
	}
	return n.send(ctx, notice.Email, subject, body)
}

func (n *SMTPNotifier) SendPasswordReset(ctx context.Context, notice service.PasswordResetNotice) error {
	data := n.templateData(notice.URL)
	subject, err := n.localize("email.passwordReset.subject", data)
	if err != nil {
		return err
	}
	body, err := n.localize("email.passwordReset.body", data)
	if err != nil {
		return err
	}
	return n.send(ctx, notice.Email, subject, body)
}

func (n *SMTPNotifier) SendEmailVerification(ctx context.Context, notice service.EmailVerificationNotice) error {
	data := n.templateData(notice.URL)
	subject, err := n.localize("email.emailVerification.subject", data)
	if err != nil {
		return err
	}
	body, err := n.localize("email.emailVerification.body", data)
	if err != nil {
		return err
	}
	return n.send(ctx, notice.Email, subject, body)
}

func (n *SMTPNotifier) send(ctx context.Context, to string, subject string, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return service.ErrInvalidInput
	}
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	if subject == "" || body == "" {
		return fmt.Errorf("%w: empty mail template", service.ErrInvalidInput)
	}
	return n.cfg.Sender.Send(ctx, MailMessage{
		To:       []string{to},
		Subject:  subject,
		TextBody: body,
	})
}

func (n *SMTPNotifier) localize(key string, data map[string]any) (string, error) {
	value := strings.TrimSpace(n.cfg.Localize(key, data))
	if value == "" || value == key {
		return "", fmt.Errorf("%w: unresolved mail template %s", service.ErrInvalidInput, key)
	}
	return value, nil
}

func (n *SMTPNotifier) templateData(url string) map[string]any {
	data := make(map[string]any, len(n.cfg.TemplateData)+1)
	for key, value := range n.cfg.TemplateData {
		data[key] = value
	}
	data["URL"] = strings.TrimSpace(url)
	return data
}
