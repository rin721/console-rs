package plugin

import (
	"context"
	"encoding/json"

	pluginpkg "github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/injection"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

const CapabilityIAMAPITokenIssue = "iam.apiToken.issue"

// DisabledAPITokenProvider 声明保留的 IAM API Token 能力，但默认拒绝实际调用。
//
// 这样插件协商可以看到能力边界和所需权限，同时在 IAM 策略未接入前不会意外暴露发 token 能力。
func DisabledAPITokenProvider() pluginpkg.Provider {
	return pluginpkg.ProviderFunc{
		Definition: protocol.Capability{
			Name:         CapabilityIAMAPITokenIssue,
			Version:      "v1",
			Scope:        protocol.CapabilityScopeSystem,
			Permissions:  []string{"api_token:create"},
			SecretPolicy: protocol.SecretPolicyOneTime,
			Description:  "Reserved IAM API token issuing capability. Disabled until explicitly wired to IAM policy.",
		},
		Handler: func(context.Context, protocol.InvokeRequest) (json.RawMessage, error) {
			return nil, pluginpkg.ErrProviderUnavailable
		},
	}
}

// RegisterProjectInjection 注册项目级上下文注入声明。
//
// 当前注入只返回一个 disabled capability descriptor，用来向远程插件明确“能力存在但默认不可用”的原因。
func RegisterProjectInjection(host *pluginpkg.Host) error {
	return host.RegisterInjectionProvider(injection.ProviderFunc{
		Definition: injection.Capability{
			Name:        "system.iam.api_token.issue",
			Version:     "v1",
			Kind:        "action",
			Permissions: []string{"api_token:create"},
			Schema: jsonObject(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"capability": map[string]any{"type": "string"},
					"enabled":    map[string]any{"type": "boolean"},
					"reason":     map[string]any{"type": "string"},
				},
				"required": []string{"capability", "enabled"},
			}),
			Description: "IAM API token issuing action exposed as a disabled, permission-scoped remote capability declaration.",
		},
		Builder: func(context.Context, injection.Request) (json.RawMessage, error) {
			return jsonObject(map[string]any{
				"capability": CapabilityIAMAPITokenIssue,
				"enabled":    false,
				"reason":     "IAM token issuing is declared but not exposed to remote plugins by default.",
			}), nil
		},
	})
}
