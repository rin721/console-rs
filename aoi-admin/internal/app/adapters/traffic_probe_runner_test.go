package adapters

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/internal/modules/system/model"
)

func TestTrafficProbeRunnerSuccessWithContentKeyword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("aoi expected marker"))
	}))
	defer server.Close()

	result := NewTrafficProbeRunner().Probe(t.Context(), probeTarget(server.URL, "expected"))
	if result.Status != model.TrafficProbeStatusHealthy {
		t.Fatalf("Status = %q, want healthy: %#v", result.Status, result)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", result.StatusCode)
	}
}

func TestTrafficProbeRunnerDetectsUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	target := probeTarget(server.URL, "")
	target.ExpectedStatusCodes = "200-399"
	result := NewTrafficProbeRunner().Probe(t.Context(), target)
	if result.Status != model.TrafficProbeStatusWarning {
		t.Fatalf("Status = %q, want warning", result.Status)
	}
	if result.Reason != "unexpected status code" {
		t.Fatalf("Reason = %q, want unexpected status code", result.Reason)
	}
}

func TestTrafficProbeRunnerDetectsMissingContentKeyword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("plain response"))
	}))
	defer server.Close()

	result := NewTrafficProbeRunner().Probe(t.Context(), probeTarget(server.URL, "expected"))
	if result.Status != model.TrafficProbeStatusWarning {
		t.Fatalf("Status = %q, want warning", result.Status)
	}
	if result.Stage != "content" {
		t.Fatalf("Stage = %q, want content", result.Stage)
	}
}

func TestTrafficProbeRunnerDetectsUnexpectedFinalHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/landing", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	target := probeTarget(server.URL+"/redirect", "")
	target.ExpectedFinalHost = "example.invalid"
	result := NewTrafficProbeRunner().Probe(t.Context(), target)
	if result.Status != model.TrafficProbeStatusCritical {
		t.Fatalf("Status = %q, want critical", result.Status)
	}
	if result.Reason != "unexpected final host" {
		t.Fatalf("Reason = %q, want unexpected final host", result.Reason)
	}
}

func TestTrafficProbeRunnerBlocksPrivateNetworkByDefault(t *testing.T) {
	target := probeTarget("http://127.0.0.1:1", "")
	target.AllowPrivateNetwork = false
	result := NewTrafficProbeRunner().Probe(t.Context(), target)
	if result.Status != model.TrafficProbeStatusCritical {
		t.Fatalf("Status = %q, want critical", result.Status)
	}
	if !strings.Contains(result.ErrorMessage, "private network address blocked") {
		t.Fatalf("ErrorMessage = %q, want private network block", result.ErrorMessage)
	}
}

func probeTarget(rawURL string, keyword string) model.TrafficProbeTarget {
	return model.TrafficProbeTarget{
		ID:                     1,
		Name:                   "test",
		URL:                    rawURL,
		Method:                 model.TrafficProbeMethodGET,
		AllowPrivateNetwork:    true,
		ExpectedContentKeyword: keyword,
		ExpectedStatusCodes:    "200-399",
		IntervalSeconds:        30,
		TimeoutSeconds:         5,
	}
}
