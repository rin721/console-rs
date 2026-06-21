package mail

import "errors"

var (
	ErrInvalidConfig  = errors.New("invalid mail config")
	ErrInvalidMessage = errors.New("invalid mail message")
	ErrConnect        = errors.New("mail connect failed")
	ErrTLSHandshake   = errors.New("mail tls handshake failed")
	ErrStartTLS       = errors.New("mail starttls failed")
	ErrAuth           = errors.New("mail auth failed")
	ErrNoop           = errors.New("mail noop failed")
)
