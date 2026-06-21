package plugin

import (
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

type InstanceContext struct {
	Now                    time.Time
	DefaultProtocolVersion string
	Source                 string
	OwnerHost              string
	LeaseTTL               time.Duration
}

type Instance struct {
	Snapshot        protocol.PluginSnapshot
	ProtocolVersion string
	Transport       string
	Endpoint        string
	Audit           protocol.AuditInfo
}

func CreatePluginApp(req protocol.RegisterRequest, ctx InstanceContext) (*Instance, error) {
	return NewPluginInstance(req, ctx)
}

func NewPluginInstance(req protocol.RegisterRequest, ctx InstanceContext) (*Instance, error) {
	metadata, err := normalizeMetadata(req.Plugin)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(metadata.SchemaVersion) == "" {
		metadata.SchemaVersion = strings.TrimSpace(ctx.DefaultProtocolVersion)
	}
	now := ctx.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	source := strings.TrimSpace(ctx.Source)
	if source == "" {
		source = "plugin-host"
	}
	leaseTTL := ctx.LeaseTTL
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}
	snapshot := protocol.PluginSnapshot{
		PluginMetadata:  metadata,
		Status:          protocol.StatusOnline,
		RuntimeStatus:   protocol.RuntimeStatusReady,
		OwnerHost:       strings.TrimSpace(ctx.OwnerHost),
		LeaseTTLSeconds: int(leaseTTL.Seconds()),
		LeaseExpiresAt:  now.Add(leaseTTL),
		RegisteredAt:    now,
		LastHeartbeatAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return &Instance{
		Snapshot:        snapshot,
		ProtocolVersion: metadata.SchemaVersion,
		Transport:       metadata.Transport,
		Endpoint:        metadata.Endpoint,
		Audit: protocol.AuditInfo{
			GeneratedAt: now,
			Source:      source,
		},
	}, nil
}
