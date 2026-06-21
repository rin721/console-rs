package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	netmail "net/mail"
	"net/textproto"
	"strings"
	"time"
)

// Message 描述一封待发送邮件。
type Message struct {
	To       []string
	Subject  string
	TextBody string
	HTMLBody string
	Headers  map[string]string
}

func buildMessage(from netmail.Address, msg Message) ([]byte, []string, error) {
	recipients, err := parseRecipients(msg.To)
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(msg.TextBody) == "" && strings.TrimSpace(msg.HTMLBody) == "" {
		return nil, nil, fmt.Errorf("%w: message body is required", ErrInvalidMessage)
	}

	var buf bytes.Buffer
	headers := map[string]string{
		"From":         from.String(),
		"To":           joinAddresses(recipients),
		"Subject":      mime.QEncoding.Encode("UTF-8", strings.TrimSpace(msg.Subject)),
		"Date":         time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version": "1.0",
	}
	for key, value := range msg.Headers {
		if err := validateHeader(key, value); err != nil {
			return nil, nil, err
		}
		headers[textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}

	switch {
	case msg.TextBody != "" && msg.HTMLBody != "":
		if err := writeMultipartAlternative(&buf, headers, msg.TextBody, msg.HTMLBody); err != nil {
			return nil, nil, err
		}
	case msg.HTMLBody != "":
		headers["Content-Type"] = `text/html; charset="UTF-8"`
		headers["Content-Transfer-Encoding"] = "8bit"
		writeHeaders(&buf, headers)
		buf.WriteString(msg.HTMLBody)
	case msg.TextBody != "":
		headers["Content-Type"] = `text/plain; charset="UTF-8"`
		headers["Content-Transfer-Encoding"] = "8bit"
		writeHeaders(&buf, headers)
		buf.WriteString(msg.TextBody)
	}

	return buf.Bytes(), recipientAddresses(recipients), nil
}

func parseRecipients(values []string) ([]netmail.Address, error) {
	recipients := make([]netmail.Address, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parsed, err := netmail.ParseAddress(value)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid recipient address: %v", ErrInvalidMessage, err)
		}
		recipients = append(recipients, *parsed)
	}
	if len(recipients) == 0 {
		return nil, fmt.Errorf("%w: recipient is required", ErrInvalidMessage)
	}
	return recipients, nil
}

func writeMultipartAlternative(buf *bytes.Buffer, headers map[string]string, textBody, htmlBody string) error {
	boundary, err := randomBoundary()
	if err != nil {
		return err
	}
	headers["Content-Type"] = `multipart/alternative; boundary="` + boundary + `"`
	writeHeaders(buf, headers)

	writer := multipart.NewWriter(buf)
	if err := writer.SetBoundary(boundary); err != nil {
		return err
	}
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", `text/plain; charset="UTF-8"`)
	textHeader.Set("Content-Transfer-Encoding", "8bit")
	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(textPart, textBody); err != nil {
		return err
	}
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", `text/html; charset="UTF-8"`)
	htmlHeader.Set("Content-Transfer-Encoding", "8bit")
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(htmlPart, htmlBody); err != nil {
		return err
	}
	return writer.Close()
}

func writeHeaders(buf *bytes.Buffer, headers map[string]string) {
	for _, key := range []string{"From", "To", "Subject", "Date", "MIME-Version", "Content-Type", "Content-Transfer-Encoding"} {
		if value, ok := headers[key]; ok {
			buf.WriteString(key)
			buf.WriteString(": ")
			buf.WriteString(value)
			buf.WriteString("\r\n")
			delete(headers, key)
		}
	}
	for key, value := range headers {
		buf.WriteString(key)
		buf.WriteString(": ")
		buf.WriteString(value)
		buf.WriteString("\r\n")
	}
	buf.WriteString("\r\n")
}

func validateHeader(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, "\r\n:") || strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%w: invalid mail header", ErrInvalidMessage)
	}
	return nil
}

func joinAddresses(addresses []netmail.Address) string {
	out := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		out = append(out, addr.String())
	}
	return strings.Join(out, ", ")
}

func recipientAddresses(addresses []netmail.Address) []string {
	out := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		out = append(out, addr.Address)
	}
	return out
}

func randomBoundary() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "mail-" + hex.EncodeToString(raw[:]), nil
}
