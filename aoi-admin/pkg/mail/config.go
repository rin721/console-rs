package mail

import (
	"fmt"
	"net"
	netmail "net/mail"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultSMTPPort    = 587
	DefaultDialTimeout = 5 * time.Second
	SecurityNone       = "none"
	SecurityStartTLS   = "starttls"
	SecurityTLS        = "tls"
)

// Config 描述 SMTP 邮件基础设施配置。
type Config struct {
	Host        string
	Port        int
	Username    string
	Password    string
	From        string
	FromName    string
	Security    string
	DialTimeout time.Duration
}

func (c *Config) applyDefaults() {
	c.Host = strings.TrimSpace(c.Host)
	c.Username = strings.TrimSpace(c.Username)
	c.From = strings.TrimSpace(c.From)
	c.FromName = strings.TrimSpace(c.FromName)
	c.Security = strings.ToLower(strings.TrimSpace(c.Security))
	if c.Security == "" {
		c.Security = SecurityNone
	}
	if c.Port == 0 {
		switch c.Security {
		case SecurityTLS:
			c.Port = 465
		case SecurityNone:
			c.Port = 25
		default:
			c.Port = DefaultSMTPPort
		}
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = DefaultDialTimeout
	}
}

func (c Config) validate() error {
	if c.Host == "" {
		return fmt.Errorf("%w: smtp host is required", ErrInvalidConfig)
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("%w: smtp port out of range", ErrInvalidConfig)
	}
	if c.From == "" {
		return fmt.Errorf("%w: sender address is required", ErrInvalidConfig)
	}
	if _, err := c.fromAddress(); err != nil {
		return fmt.Errorf("%w: invalid sender address: %v", ErrInvalidConfig, err)
	}
	switch c.Security {
	case SecurityNone, SecurityStartTLS, SecurityTLS:
	default:
		return fmt.Errorf("%w: smtp security must be one of none, starttls, tls", ErrInvalidConfig)
	}
	return nil
}

func (c Config) address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c Config) fromAddress() (netmail.Address, error) {
	addr := netmail.Address{Name: c.FromName, Address: c.From}
	parsed, err := netmail.ParseAddress(addr.String())
	if err != nil {
		return netmail.Address{}, err
	}
	return *parsed, nil
}
