package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
)

const trafficProbeMaxBodyBytes = 1024 * 1024

type TrafficProbeRunner struct {
	Resolver *net.Resolver
}

func NewTrafficProbeRunner() TrafficProbeRunner {
	return TrafficProbeRunner{Resolver: net.DefaultResolver}
}

func (r TrafficProbeRunner) Probe(ctx context.Context, target model.TrafficProbeTarget) model.TrafficProbeResult {
	start := time.Now()
	timeout := time.Duration(target.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parsed, err := url.Parse(target.URL)
	if err != nil {
		return failedTrafficProbe(target, "url", "invalid target url", err, start)
	}
	recorder := &trafficProbeTrace{}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           r.dialContext(target, recorder),
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			recorder.addRedirect(req.URL.String())
			if len(via) >= 5 {
				return errors.New("redirect limit exceeded")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(probeCtx, recorder.trace()), target.Method, parsed.String(), nil)
	if err != nil {
		return failedTrafficProbe(target, "request", "invalid probe request", err, start)
	}
	req.Header.Set("User-Agent", "aoi-admin-traffic-probe/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return failedTrafficProbeWithTrace(target, recorder, "request", "probe request failed", err, start)
	}
	defer resp.Body.Close()

	result := model.TrafficProbeResult{
		DNSIPs:            strings.Join(recorder.ips(), ","),
		DNSDurationMs:     recorder.dnsDurationMs(),
		ConnectDurationMs: recorder.connectDurationMs(),
		TLSDurationMs:     recorder.tlsDurationMs(),
		TTFBMs:            recorder.ttfbMs(),
		TotalDurationMs:   time.Since(start).Milliseconds(),
		FinalURL:          resp.Request.URL.String(),
		StatusCode:        resp.StatusCode,
	}
	applyTLSSummary(&result, resp.TLS)
	evidence := map[string]any{
		"dnsIps":      recorder.ips(),
		"redirects":   recorder.redirects(),
		"finalUrl":    result.FinalURL,
		"statusCode":  resp.StatusCode,
		"tlsSubject":  result.TLSSubject,
		"tlsIssuer":   result.TLSIssuer,
		"fingerprint": result.TLSFingerprintSHA256,
	}
	if issue := validateTrafficProbeResponse(target, resp, &result); issue != nil {
		result.Status = issue.status
		result.Severity = issue.severity
		result.Reason = issue.reason
		result.Stage = issue.stage
		result.ErrorMessage = issue.message
		evidence["issue"] = issue.reason
		result.EvidenceJSON = evidenceJSON(evidence)
		return result
	}
	if strings.TrimSpace(target.ExpectedContentKeyword) != "" && target.Method != model.TrafficProbeMethodHEAD {
		body, err := io.ReadAll(io.LimitReader(resp.Body, trafficProbeMaxBodyBytes))
		if err != nil {
			return failedTrafficProbeWithTrace(target, recorder, "body", "read response body failed", err, start)
		}
		if !bytes.Contains(body, []byte(target.ExpectedContentKeyword)) {
			result.Status = model.TrafficProbeStatusWarning
			result.Severity = model.TrafficProbeSeverityMedium
			result.Reason = "content keyword missing"
			result.Stage = "content"
			result.ErrorMessage = "expected content keyword was not found"
			result.EvidenceJSON = evidenceJSON(evidence)
			return result
		}
	}
	result.Status = model.TrafficProbeStatusHealthy
	result.Severity = model.TrafficProbeSeverityOK
	result.Reason = "probe healthy"
	result.Stage = "complete"
	result.EvidenceJSON = evidenceJSON(evidence)
	return result
}

func (r TrafficProbeRunner) dialContext(target model.TrafficProbeTarget, recorder *trafficProbeTrace) func(context.Context, string, string) (net.Conn, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := &net.Dialer{}
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		dnsStart := time.Now()
		addrs, err := resolver.LookupIPAddr(ctx, host)
		recorder.addDNSDuration(time.Since(dnsStart))
		if err != nil {
			return nil, err
		}
		allowedCIDRs := parseCIDRs(target.ExpectedIPCIDRs)
		for _, addr := range addrs {
			ip := addr.IP
			recorder.addIP(ip.String())
			if !target.AllowPrivateNetwork && unsafeProbeIP(ip) {
				return nil, fmt.Errorf("private network address blocked: %s", ip.String())
			}
			if len(allowedCIDRs) > 0 && !ipInAnyCIDR(ip, allowedCIDRs) {
				continue
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}
		if len(allowedCIDRs) > 0 {
			return nil, fmt.Errorf("resolved IPs do not match expected CIDRs for %s", host)
		}
		return nil, fmt.Errorf("no resolved IP address for %s", host)
	}
}

type trafficProbeIssue struct {
	status   string
	severity string
	reason   string
	stage    string
	message  string
}

func validateTrafficProbeResponse(target model.TrafficProbeTarget, resp *http.Response, result *model.TrafficProbeResult) *trafficProbeIssue {
	if !statusAllowed(resp.StatusCode, target.ExpectedStatusCodes) {
		return &trafficProbeIssue{status: model.TrafficProbeStatusWarning, severity: model.TrafficProbeSeverityMedium, reason: "unexpected status code", stage: "status", message: http.StatusText(resp.StatusCode)}
	}
	if expectedHost := strings.ToLower(strings.TrimSpace(target.ExpectedFinalHost)); expectedHost != "" {
		if strings.ToLower(resp.Request.URL.Hostname()) != expectedHost {
			return &trafficProbeIssue{status: model.TrafficProbeStatusCritical, severity: model.TrafficProbeSeverityHigh, reason: "unexpected final host", stage: "redirect", message: "final host does not match expected host"}
		}
	}
	expectedFingerprint := normalizeFingerprint(target.ExpectedTLSFingerprint)
	if expectedFingerprint != "" {
		if normalizeFingerprint(result.TLSFingerprintSHA256) != expectedFingerprint {
			return &trafficProbeIssue{status: model.TrafficProbeStatusCritical, severity: model.TrafficProbeSeverityHigh, reason: "tls fingerprint mismatch", stage: "tls", message: "leaf certificate fingerprint changed"}
		}
	}
	return nil
}

func failedTrafficProbe(target model.TrafficProbeTarget, stage string, reason string, err error, start time.Time) model.TrafficProbeResult {
	return failedTrafficProbeWithTrace(target, &trafficProbeTrace{}, stage, reason, err, start)
}

func failedTrafficProbeWithTrace(target model.TrafficProbeTarget, recorder *trafficProbeTrace, stage string, reason string, err error, start time.Time) model.TrafficProbeResult {
	return model.TrafficProbeResult{
		TargetID:          target.ID,
		TargetName:        target.Name,
		URL:               target.URL,
		Method:            target.Method,
		Status:            model.TrafficProbeStatusCritical,
		Severity:          model.TrafficProbeSeverityHigh,
		Reason:            reason,
		Stage:             stage,
		ErrorMessage:      errString(err),
		DNSIPs:            strings.Join(recorder.ips(), ","),
		DNSDurationMs:     recorder.dnsDurationMs(),
		ConnectDurationMs: recorder.connectDurationMs(),
		TLSDurationMs:     recorder.tlsDurationMs(),
		TTFBMs:            recorder.ttfbMs(),
		TotalDurationMs:   time.Since(start).Milliseconds(),
		EvidenceJSON: evidenceJSON(map[string]any{
			"dnsIps": recorder.ips(),
			"error":  errString(err),
			"stage":  stage,
		}),
	}
}

type trafficProbeTrace struct {
	mu              sync.Mutex
	dnsDuration     time.Duration
	connectDuration time.Duration
	tlsDuration     time.Duration
	ttfbDuration    time.Duration
	connectStart    time.Time
	tlsStart        time.Time
	requestStart    time.Time
	dnsIPs          []string
	redirectURLs    []string
	seenIP          map[string]struct{}
}

func (t *trafficProbeTrace) trace() *httptrace.ClientTrace {
	t.requestStart = time.Now()
	return &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) {
			t.mu.Lock()
			t.connectStart = time.Now()
			t.mu.Unlock()
		},
		ConnectDone: func(_, _ string, _ error) {
			t.mu.Lock()
			if !t.connectStart.IsZero() {
				t.connectDuration += time.Since(t.connectStart)
			}
			t.mu.Unlock()
		},
		TLSHandshakeStart: func() {
			t.mu.Lock()
			t.tlsStart = time.Now()
			t.mu.Unlock()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			t.mu.Lock()
			if !t.tlsStart.IsZero() {
				t.tlsDuration += time.Since(t.tlsStart)
			}
			t.mu.Unlock()
		},
		GotFirstResponseByte: func() {
			t.mu.Lock()
			if !t.requestStart.IsZero() && t.ttfbDuration == 0 {
				t.ttfbDuration = time.Since(t.requestStart)
			}
			t.mu.Unlock()
		},
	}
}

func (t *trafficProbeTrace) addDNSDuration(value time.Duration) {
	t.mu.Lock()
	t.dnsDuration += value
	t.mu.Unlock()
}

func (t *trafficProbeTrace) addIP(value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.seenIP == nil {
		t.seenIP = map[string]struct{}{}
	}
	if _, ok := t.seenIP[value]; ok {
		return
	}
	t.seenIP[value] = struct{}{}
	t.dnsIPs = append(t.dnsIPs, value)
}

func (t *trafficProbeTrace) addRedirect(value string) {
	t.mu.Lock()
	t.redirectURLs = append(t.redirectURLs, value)
	t.mu.Unlock()
}

func (t *trafficProbeTrace) ips() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.dnsIPs...)
}

func (t *trafficProbeTrace) redirects() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.redirectURLs...)
}

func (t *trafficProbeTrace) dnsDurationMs() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.dnsDuration.Milliseconds()
}

func (t *trafficProbeTrace) connectDurationMs() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connectDuration.Milliseconds()
}

func (t *trafficProbeTrace) tlsDurationMs() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tlsDuration.Milliseconds()
}

func (t *trafficProbeTrace) ttfbMs() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ttfbDuration.Milliseconds()
}

func applyTLSSummary(result *model.TrafficProbeResult, state *tls.ConnectionState) {
	if state == nil || len(state.PeerCertificates) == 0 {
		return
	}
	cert := state.PeerCertificates[0]
	result.TLSSubject = cert.Subject.String()
	result.TLSIssuer = cert.Issuer.String()
	notAfter := cert.NotAfter.UTC()
	result.TLSNotAfter = &notAfter
	sum := sha256.Sum256(cert.Raw)
	result.TLSFingerprintSHA256 = hex.EncodeToString(sum[:])
}

func unsafeProbeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

func parseCIDRs(value string) []*net.IPNet {
	parts := strings.Split(value, ",")
	out := make([]*net.IPNet, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if ip := net.ParseIP(part); ip != nil {
			if ip.To4() != nil {
				part += "/32"
			} else {
				part += "/128"
			}
		}
		_, cidr, err := net.ParseCIDR(part)
		if err == nil {
			out = append(out, cidr)
		}
	}
	return out
}

func ipInAnyCIDR(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func statusAllowed(status int, spec string) bool {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		spec = "200-399"
	}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if left, right, ok := strings.Cut(part, "-"); ok {
			min, minErr := strconv.Atoi(strings.TrimSpace(left))
			max, maxErr := strconv.Atoi(strings.TrimSpace(right))
			if minErr == nil && maxErr == nil && status >= min && status <= max {
				return true
			}
			continue
		}
		code, err := strconv.Atoi(part)
		if err == nil && code == status {
			return true
		}
	}
	return false
}

func normalizeFingerprint(value string) string {
	replacer := strings.NewReplacer(":", "", " ", "", "-", "")
	return strings.ToLower(strings.TrimSpace(replacer.Replace(value)))
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func evidenceJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
