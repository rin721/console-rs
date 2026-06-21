package model

import "time"

const (
	TrafficProbeMethodGET  = "GET"
	TrafficProbeMethodHEAD = "HEAD"

	TrafficProbeStatusPending  = "pending"
	TrafficProbeStatusHealthy  = "healthy"
	TrafficProbeStatusWarning  = "warning"
	TrafficProbeStatusCritical = "critical"

	TrafficProbeSeverityOK       = "ok"
	TrafficProbeSeverityLow      = "low"
	TrafficProbeSeverityMedium   = "medium"
	TrafficProbeSeverityHigh     = "high"
	TrafficProbeSeverityCritical = "critical"

	TrafficHijackEventStateOpen     = "open"
	TrafficHijackEventStateResolved = "resolved"

	TrafficAlertChannelEvent = "event"
	TrafficAlertChannelDebug = "debug"
	TrafficAlertChannelEmail = "email"
)

type TrafficProbeTarget struct {
	ID                     int64      `gorm:"column:id;primaryKey" json:"id,string"`
	Name                   string     `gorm:"column:name;size:128;not null" json:"name"`
	URL                    string     `gorm:"column:url;type:text;not null" json:"url"`
	Method                 string     `gorm:"column:http_method;size:8;not null" json:"method"`
	Enabled                bool       `gorm:"column:enabled;not null" json:"enabled"`
	IntervalSeconds        int        `gorm:"column:interval_seconds;not null" json:"intervalSeconds"`
	TimeoutSeconds         int        `gorm:"column:timeout_seconds;not null" json:"timeoutSeconds"`
	ExpectedStatusCodes    string     `gorm:"column:expected_status_codes;size:128;not null" json:"expectedStatusCodes"`
	ExpectedFinalHost      string     `gorm:"column:expected_final_host;size:255;not null" json:"expectedFinalHost"`
	ExpectedContentKeyword string     `gorm:"column:expected_content_keyword;type:text;not null" json:"expectedContentKeyword"`
	ExpectedIPCIDRs        string     `gorm:"column:expected_ip_cidrs;type:text;not null" json:"expectedIpCidrs"`
	ExpectedTLSFingerprint string     `gorm:"column:expected_tls_fingerprint;size:128;not null" json:"expectedTlsFingerprint"`
	AllowPrivateNetwork    bool       `gorm:"column:allow_private_network;not null" json:"allowPrivateNetwork"`
	AlertChannels          string     `gorm:"column:alert_channels;size:128;not null" json:"alertChannels"`
	EmailRecipients        string     `gorm:"column:email_recipients;type:text;not null" json:"emailRecipients"`
	LastStatus             string     `gorm:"column:last_status;size:32;not null" json:"lastStatus"`
	LastSeverity           string     `gorm:"column:last_severity;size:32;not null" json:"lastSeverity"`
	LastReason             string     `gorm:"column:last_reason;type:text;not null" json:"lastReason"`
	LastProbedAt           *time.Time `gorm:"column:last_probed_at" json:"lastProbedAt,omitempty"`
	NextProbeAt            *time.Time `gorm:"column:next_probe_at;index" json:"nextProbeAt,omitempty"`
	CreatedAt              time.Time  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt              time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	DeletedAt              *time.Time `gorm:"column:deleted_at;index" json:"-"`
}

func (TrafficProbeTarget) TableName() string { return "system_traffic_probe_targets" }

type TrafficProbeResult struct {
	ID                   int64      `gorm:"column:id;primaryKey" json:"id,string"`
	TargetID             int64      `gorm:"column:target_id;not null;index" json:"targetId,string"`
	TargetName           string     `gorm:"column:target_name;size:128;not null" json:"targetName"`
	URL                  string     `gorm:"column:url;type:text;not null" json:"url"`
	Method               string     `gorm:"column:http_method;size:8;not null" json:"method"`
	Status               string     `gorm:"column:status;size:32;not null;index" json:"status"`
	Severity             string     `gorm:"column:severity;size:32;not null;index" json:"severity"`
	Reason               string     `gorm:"column:reason;size:255;not null" json:"reason"`
	Stage                string     `gorm:"column:stage;size:64;not null" json:"stage"`
	ErrorMessage         string     `gorm:"column:error_message;type:text;not null" json:"errorMessage"`
	DNSIPs               string     `gorm:"column:dns_ips;type:text;not null" json:"dnsIps"`
	FinalURL             string     `gorm:"column:final_url;type:text;not null" json:"finalUrl"`
	StatusCode           int        `gorm:"column:status_code;not null" json:"statusCode"`
	TLSSubject           string     `gorm:"column:tls_subject;size:255;not null" json:"tlsSubject"`
	TLSIssuer            string     `gorm:"column:tls_issuer;size:255;not null" json:"tlsIssuer"`
	TLSNotAfter          *time.Time `gorm:"column:tls_not_after" json:"tlsNotAfter,omitempty"`
	TLSFingerprintSHA256 string     `gorm:"column:tls_fingerprint_sha256;size:128;not null" json:"tlsFingerprintSha256"`
	DNSDurationMs        int64      `gorm:"column:dns_duration_ms;not null" json:"dnsDurationMs"`
	ConnectDurationMs    int64      `gorm:"column:connect_duration_ms;not null" json:"connectDurationMs"`
	TLSDurationMs        int64      `gorm:"column:tls_duration_ms;not null" json:"tlsDurationMs"`
	TTFBMs               int64      `gorm:"column:ttfb_ms;not null" json:"ttfbMs"`
	TotalDurationMs      int64      `gorm:"column:total_duration_ms;not null" json:"totalDurationMs"`
	EvidenceJSON         string     `gorm:"column:evidence_json;type:text;not null" json:"evidenceJson"`
	CreatedAt            time.Time  `gorm:"column:created_at;not null;index" json:"createdAt"`
}

func (TrafficProbeResult) TableName() string { return "system_traffic_probe_results" }

type TrafficHijackEvent struct {
	ID                 int64      `gorm:"column:id;primaryKey" json:"id,string"`
	TargetID           int64      `gorm:"column:target_id;not null;index" json:"targetId,string"`
	TargetName         string     `gorm:"column:target_name;size:128;not null" json:"targetName"`
	Reason             string     `gorm:"column:reason;size:255;not null" json:"reason"`
	Severity           string     `gorm:"column:severity;size:32;not null;index" json:"severity"`
	State              string     `gorm:"column:state;size:32;not null;index" json:"state"`
	EvidenceHash       string     `gorm:"column:evidence_hash;size:64;not null;index" json:"evidenceHash"`
	EvidenceJSON       string     `gorm:"column:evidence_json;type:text;not null" json:"evidenceJson"`
	FirstSeenAt        time.Time  `gorm:"column:first_seen_at;not null" json:"firstSeenAt"`
	LastSeenAt         time.Time  `gorm:"column:last_seen_at;not null" json:"lastSeenAt"`
	ResolvedAt         *time.Time `gorm:"column:resolved_at" json:"resolvedAt,omitempty"`
	Occurrences        int        `gorm:"column:occurrences;not null" json:"occurrences"`
	NotificationStatus string     `gorm:"column:notification_status;size:64;not null" json:"notificationStatus"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (TrafficHijackEvent) TableName() string { return "system_traffic_hijack_events" }

type TrafficHijackOverview struct {
	TotalTargets    int                  `json:"totalTargets"`
	EnabledTargets  int                  `json:"enabledTargets"`
	HealthyTargets  int                  `json:"healthyTargets"`
	WarningTargets  int                  `json:"warningTargets"`
	CriticalTargets int                  `json:"criticalTargets"`
	OpenEvents      int                  `json:"openEvents"`
	LastProbeAt     *time.Time           `json:"lastProbeAt,omitempty"`
	RecentResults   []TrafficProbeResult `json:"recentResults"`
	RecentEvents    []TrafficHijackEvent `json:"recentEvents"`
}

type TrafficProbeResultPage struct {
	Items         []TrafficProbeResult `json:"items"`
	Limit         int                  `json:"limit"`
	NextCursor    int64                `json:"nextCursor,string,omitempty"`
	StorageStatus string               `json:"storageStatus"`
}

type TrafficProbeResultFilter struct {
	TargetID int64
	Limit    int
	Cursor   int64
}

type TrafficHijackEventPage struct {
	Items         []TrafficHijackEvent `json:"items"`
	Page          int                  `json:"page"`
	PageSize      int                  `json:"pageSize"`
	StorageStatus string               `json:"storageStatus"`
	Total         int64                `json:"total"`
}

type TrafficHijackEventFilter struct {
	TargetID int64
	Severity string
	State    string
	Page     int
	PageSize int
}

type TrafficHijackStreamEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}
