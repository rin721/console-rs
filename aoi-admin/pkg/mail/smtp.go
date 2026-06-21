package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

// Sender 是邮件投递端口。
type Sender interface {
	Send(context.Context, Message) error
}

// Checker 是不投递邮件的 SMTP 可用性检查端口。
type Checker interface {
	Check(context.Context) error
}

// SMTPSender 使用 SMTP 协议发送邮件并检查连接。
type SMTPSender struct {
	cfg       Config
	tlsConfig *tls.Config
}

func NewSMTP(cfg Config) (*SMTPSender, error) {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &SMTPSender{cfg: cfg}, nil
}

func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	from, err := s.cfg.fromAddress()
	if err != nil {
		return err
	}
	data, recipients, err := buildMessage(from, msg)
	if err != nil {
		return err
	}
	client, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := s.prepare(ctx, client); err != nil {
		return err
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (s *SMTPSender) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := s.prepare(ctx, client); err != nil {
		return err
	}
	if err := client.Noop(); err != nil {
		return fmt.Errorf("%w: %v", ErrNoop, err)
	}
	return client.Quit()
}

func (s *SMTPSender) connect(ctx context.Context) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: s.cfg.DialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", s.cfg.address())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnect, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else if s.cfg.DialTimeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(s.cfg.DialTimeout))
	}
	if s.cfg.Security == SecurityTLS {
		tlsConn := tls.Client(conn, s.effectiveTLSConfig())
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("%w: %v", ErrTLSHandshake, err)
		}
		conn = tlsConn
	}
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (s *SMTPSender) prepare(ctx context.Context, client *smtp.Client) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := client.Hello("localhost"); err != nil {
		return err
	}
	if s.cfg.Security == SecurityStartTLS {
		if err := client.StartTLS(s.effectiveTLSConfig()); err != nil {
			return fmt.Errorf("%w: %v", ErrStartTLS, err)
		}
	}
	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("%w: %v", ErrAuth, err)
		}
	}
	return ctx.Err()
}

func (s *SMTPSender) effectiveTLSConfig() *tls.Config {
	tlsCfg := s.tlsConfig
	if tlsCfg == nil {
		return &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	}
	tlsCfg = tlsCfg.Clone()
	if tlsCfg.ServerName == "" {
		tlsCfg.ServerName = s.cfg.Host
	}
	if tlsCfg.MinVersion == 0 {
		tlsCfg.MinVersion = tls.VersionTLS12
	}
	return tlsCfg
}
