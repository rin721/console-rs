package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/rei0721/go-scaffold/internal/modules/iam/service"
	mailpkg "github.com/rei0721/go-scaffold/pkg/mail"
)

func TestSMTPNotifierRendersInvitationPasswordResetAndEmailVerification(t *testing.T) {
	sender := &captureMailSender{}
	notifier, err := NewSMTPNotifier(SMTPNotifierConfig{
		Sender:       sender,
		TemplateData: map[string]any{"ProductName": "Aoi"},
		Localize: func(key string, data map[string]any) string {
			patterns := map[string]string{
				"email.invitation.subject":        "Invite to {{.ProductName}}",
				"email.invitation.body":           "Join {{.ProductName}} at {{.URL}}",
				"email.passwordReset.subject":     "Reset {{.ProductName}} password",
				"email.passwordReset.body":        "Reset at {{.URL}}",
				"email.emailVerification.subject": "Verify {{.ProductName}} email",
				"email.emailVerification.body":    "Verify at {{.URL}}",
			}
			rendered, err := mailpkg.RenderText(patterns[key], data)
			if err != nil {
				t.Fatalf("render %s: %v", key, err)
			}
			return rendered
		},
	})
	if err != nil {
		t.Fatalf("NewSMTPNotifier() failed: %v", err)
	}
	if err := notifier.SendInvitation(context.Background(), service.InvitationNotice{Email: "user@example.com", URL: "https://example.com/invite"}); err != nil {
		t.Fatalf("SendInvitation() failed: %v", err)
	}
	if got := sender.last; got.Subject != "Invite to Aoi" || got.TextBody != "Join Aoi at https://example.com/invite" {
		t.Fatalf("unexpected invitation message: %#v", got)
	}
	if err := notifier.SendPasswordReset(context.Background(), service.PasswordResetNotice{Email: "user@example.com", URL: "https://example.com/reset"}); err != nil {
		t.Fatalf("SendPasswordReset() failed: %v", err)
	}
	if got := sender.last; got.Subject != "Reset Aoi password" || got.TextBody != "Reset at https://example.com/reset" {
		t.Fatalf("unexpected password reset message: %#v", got)
	}
	if err := notifier.SendEmailVerification(context.Background(), service.EmailVerificationNotice{Email: "user@example.com", URL: "https://example.com/verify"}); err != nil {
		t.Fatalf("SendEmailVerification() failed: %v", err)
	}
	if got := sender.last; got.Subject != "Verify Aoi email" || got.TextBody != "Verify at https://example.com/verify" {
		t.Fatalf("unexpected email verification message: %#v", got)
	}
}

func TestSMTPNotifierPropagatesSenderError(t *testing.T) {
	wantErr := errors.New("smtp down")
	notifier, err := NewSMTPNotifier(SMTPNotifierConfig{
		Sender: &captureMailSender{err: wantErr},
		Localize: func(string, map[string]any) string {
			return "body"
		},
	})
	if err != nil {
		t.Fatalf("NewSMTPNotifier() failed: %v", err)
	}
	if err := notifier.SendInvitation(context.Background(), service.InvitationNotice{Email: "user@example.com", URL: "https://example.com/invite"}); !errors.Is(err, wantErr) {
		t.Fatalf("SendInvitation() error = %v, want %v", err, wantErr)
	}
}

func TestSMTPNotifierRejectsUnresolvedTemplate(t *testing.T) {
	notifier, err := NewSMTPNotifier(SMTPNotifierConfig{
		Sender: &captureMailSender{},
		Localize: func(key string, _ map[string]any) string {
			return key
		},
	})
	if err != nil {
		t.Fatalf("NewSMTPNotifier() failed: %v", err)
	}
	if err := notifier.SendPasswordReset(context.Background(), service.PasswordResetNotice{Email: "user@example.com", URL: "https://example.com/reset"}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("SendPasswordReset() error = %v, want ErrInvalidInput", err)
	}
}

type captureMailSender struct {
	last MailMessage
	err  error
}

func (s *captureMailSender) Send(_ context.Context, msg MailMessage) error {
	s.last = msg
	return s.err
}
