package mail

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	netmail "net/mail"
	"strings"
	"testing"
	"time"
)

func TestNewSMTPValidatesConfig(t *testing.T) {
	if _, err := NewSMTP(Config{Port: 587, From: "noreply@example.com"}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("missing host error = %v, want ErrInvalidConfig", err)
	}
	if _, err := NewSMTP(Config{Host: "smtp.example.com", From: "bad"}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("bad from error = %v, want ErrInvalidConfig", err)
	}
	sender, err := NewSMTP(Config{Host: "smtp.example.com", From: "noreply@example.com"})
	if err != nil {
		t.Fatalf("NewSMTP() failed: %v", err)
	}
	if sender.cfg.Port != 25 || sender.cfg.Security != SecurityNone || sender.cfg.DialTimeout != DefaultDialTimeout {
		t.Fatalf("defaults not applied: %#v", sender.cfg)
	}
}

func TestBuildMessageEncodesHeadersAndMultipart(t *testing.T) {
	from := netmail.Address{Name: "Aoi Admin", Address: "noreply@example.com"}
	raw, recipients, err := buildMessage(from, Message{
		To:       []string{"User <user@example.com>"},
		Subject:  "密码重置",
		TextBody: "Reset: https://example.com/reset",
		HTMLBody: "<p>Reset</p>",
		Headers:  map[string]string{"X-Trace-ID": "abc123"},
	})
	if err != nil {
		t.Fatalf("buildMessage() failed: %v", err)
	}
	text := string(raw)
	if len(recipients) != 1 || recipients[0] != "user@example.com" {
		t.Fatalf("unexpected recipients: %#v", recipients)
	}
	for _, want := range []string{
		"From: \"Aoi Admin\" <noreply@example.com>",
		"To: \"User\" <user@example.com>",
		"Subject: =?UTF-8?",
		"Content-Type: multipart/alternative;",
		`Content-Type: text/plain; charset="UTF-8"`,
		`Content-Type: text/html; charset="UTF-8"`,
		"X-Trace-Id: abc123",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("message missing %q:\n%s", want, text)
		}
	}
}

func TestBuildMessageRejectsInvalidInput(t *testing.T) {
	from := netmail.Address{Address: "noreply@example.com"}
	if _, _, err := buildMessage(from, Message{To: []string{"bad"}, TextBody: "body"}); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("bad recipient error = %v, want ErrInvalidMessage", err)
	}
	if _, _, err := buildMessage(from, Message{To: []string{"user@example.com"}, TextBody: "body", Headers: map[string]string{"X-Bad": "a\r\nb"}}); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("bad header error = %v, want ErrInvalidMessage", err)
	}
}

func TestTemplateRender(t *testing.T) {
	rendered, err := (Template{
		Subject:  "Welcome {{.ProductName}}",
		TextBody: "Open {{.URL}}",
		HTMLBody: "<a href=\"{{.URL}}\">Open</a>",
	}).Render(map[string]any{"ProductName": "Aoi", "URL": "https://example.com"})
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if rendered.Subject != "Welcome Aoi" || rendered.TextBody != "Open https://example.com" {
		t.Fatalf("unexpected rendered template: %#v", rendered)
	}
	if _, err := RenderText("{{.Missing}}", map[string]any{}); err == nil {
		t.Fatal("RenderText() should reject missing keys")
	}
}

func TestSMTPSenderSend(t *testing.T) {
	server := startFakeSMTP(t, fakeSMTPOptions{})
	sender, err := NewSMTP(Config{Host: "localhost", Port: server.port, From: "noreply@example.com", FromName: "Aoi Admin"})
	if err != nil {
		t.Fatalf("NewSMTP() failed: %v", err)
	}
	if err := sender.Send(context.Background(), Message{
		To:       []string{"user@example.com"},
		Subject:  "Invite",
		TextBody: "Join https://example.com/invite",
	}); err != nil {
		t.Fatalf("Send() failed: %v", err)
	}
	message := server.message(t)
	if !strings.Contains(message, "Join https://example.com/invite") {
		t.Fatalf("sent message missing body:\n%s", message)
	}
	server.assertEvent(t, "MAIL FROM:<noreply@example.com>")
	server.assertEvent(t, "RCPT TO:<user@example.com>")
}

func TestSMTPCheckStartTLSAndAuthDoesNotSendMail(t *testing.T) {
	serverTLS, clientTLS := testTLSConfig(t)
	server := startFakeSMTP(t, fakeSMTPOptions{startTLS: true, auth: true, tlsConfig: serverTLS})
	sender, err := NewSMTP(Config{
		Host:     "localhost",
		Port:     server.port,
		Username: "smtp-user",
		Password: "smtp-pass",
		From:     "noreply@example.com",
		Security: SecurityStartTLS,
	})
	if err != nil {
		t.Fatalf("NewSMTP() failed: %v", err)
	}
	sender.tlsConfig = clientTLS
	if err := sender.Check(context.Background()); err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
	for _, want := range []string{"STARTTLS", "AUTH:smtp-user:smtp-pass", "NOOP"} {
		server.assertEvent(t, want)
	}
	select {
	case msg := <-server.messages:
		t.Fatalf("Check() should not send DATA, got %q", msg)
	default:
	}
}

func TestSMTPCheckImplicitTLSAndAuthDoesNotSendMail(t *testing.T) {
	serverTLS, clientTLS := testTLSConfig(t)
	server := startFakeSMTP(t, fakeSMTPOptions{implicitTLS: true, auth: true, tlsConfig: serverTLS})
	sender, err := NewSMTP(Config{
		Host:     "localhost",
		Port:     server.port,
		Username: "smtp-user",
		Password: "smtp-pass",
		From:     "noreply@example.com",
		Security: SecurityTLS,
	})
	if err != nil {
		t.Fatalf("NewSMTP() failed: %v", err)
	}
	sender.tlsConfig = clientTLS
	if err := sender.Check(context.Background()); err != nil {
		t.Fatalf("Check() failed: %v", err)
	}
	for _, want := range []string{"AUTH:smtp-user:smtp-pass", "NOOP"} {
		server.assertEvent(t, want)
	}
	select {
	case msg := <-server.messages:
		t.Fatalf("Check() should not send DATA, got %q", msg)
	default:
	}
}

type fakeSMTPOptions struct {
	startTLS    bool
	implicitTLS bool
	auth        bool
	tlsConfig   *tls.Config
}

type fakeSMTPServer struct {
	t        *testing.T
	listener net.Listener
	port     int
	events   chan string
	messages chan string
	options  fakeSMTPOptions
}

func startFakeSMTP(t *testing.T, options fakeSMTPOptions) *fakeSMTPServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake smtp: %v", err)
	}
	server := &fakeSMTPServer{
		t:        t,
		listener: listener,
		port:     listener.Addr().(*net.TCPAddr).Port,
		events:   make(chan string, 32),
		messages: make(chan string, 4),
		options:  options,
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})
	go server.accept()
	return server
}

func (s *fakeSMTPServer) accept() {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	s.serve(conn)
}

func (s *fakeSMTPServer) serve(conn net.Conn) {
	tlsActive := false
	if s.options.implicitTLS {
		tlsConn := tls.Server(conn, s.options.tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		conn = tlsConn
		tlsActive = true
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writeSMTPLine(writer, "220 localhost ESMTP")
	inData := false
	var data strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				s.messages <- data.String()
				data.Reset()
				inData = false
				writeSMTPLine(writer, "250 queued")
				continue
			}
			data.WriteString(line)
			data.WriteString("\n")
			continue
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO ") || strings.HasPrefix(upper, "HELO "):
			writeSMTPLine(writer, "250-localhost")
			if s.options.startTLS && !tlsActive {
				writeSMTPLine(writer, "250-STARTTLS")
			}
			if s.options.auth && tlsActive {
				writeSMTPLine(writer, "250-AUTH PLAIN")
			}
			writeSMTPLine(writer, "250 OK")
		case upper == "STARTTLS":
			s.events <- "STARTTLS"
			writeSMTPLine(writer, "220 ready")
			tlsConn := tls.Server(conn, s.options.tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
			writer = bufio.NewWriter(conn)
			tlsActive = true
		case strings.HasPrefix(upper, "AUTH PLAIN "):
			payload := strings.TrimSpace(line[len("AUTH PLAIN "):])
			raw, _ := base64.StdEncoding.DecodeString(payload)
			parts := strings.Split(string(raw), "\x00")
			if len(parts) == 3 {
				s.events <- "AUTH:" + parts[1] + ":" + parts[2]
			} else {
				s.events <- "AUTH"
			}
			writeSMTPLine(writer, "235 authenticated")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			s.events <- line
			writeSMTPLine(writer, "250 sender ok")
		case strings.HasPrefix(upper, "RCPT TO:"):
			s.events <- line
			writeSMTPLine(writer, "250 recipient ok")
		case upper == "DATA":
			inData = true
			writeSMTPLine(writer, "354 go ahead")
		case upper == "NOOP":
			s.events <- "NOOP"
			writeSMTPLine(writer, "250 ok")
		case upper == "QUIT":
			writeSMTPLine(writer, "221 bye")
			return
		default:
			writeSMTPLine(writer, "250 ok")
		}
	}
}

func writeSMTPLine(writer *bufio.Writer, line string) {
	_, _ = writer.WriteString(line + "\r\n")
	_ = writer.Flush()
}

func (s *fakeSMTPServer) message(t *testing.T) string {
	t.Helper()
	select {
	case msg := <-s.messages:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for smtp message")
	}
	return ""
}

func (s *fakeSMTPServer) assertEvent(t *testing.T, want string) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case got := <-s.events:
			if got == want {
				return
			}
		case <-timeout:
			t.Fatalf("timeout waiting for smtp event %q", want)
		}
	}
}

func testTLSConfig(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate tls key: %v", err)
	}
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create tls cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("parse tls keypair: %v", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}, &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
}
