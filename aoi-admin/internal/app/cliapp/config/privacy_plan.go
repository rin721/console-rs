package config

var coreSecretPaths = []string{
	"auth.signing_key",
	"auth.refresh_token_pepper",
	"auth.mfa_secret_key",
}

const (
	privacyActionForceFile      = "force_file"
	privacyActionRuntimeEnvOnly = "runtime_env_only"
	privacyActionSkip           = "skip"
)

// PrivacyPersistPlan 汇总一次隐私配置向导中选择的三类持久化动作。
//
// FileUpdates 走普通配置写回；ForceFileUpdates 会覆盖 env 占位符；RuntimeEnvOnlyPaths 只校验环境变量并恢复 env 管理。
type PrivacyPersistPlan struct {
	FileUpdates         map[string]string
	ForceFileUpdates    map[string]string
	RuntimeEnvOnlyPaths []string
}

func newPrivacyPersistPlan() PrivacyPersistPlan {
	return PrivacyPersistPlan{
		FileUpdates:      map[string]string{},
		ForceFileUpdates: map[string]string{},
	}
}

// HasChanges 判断向导是否产生需要落盘或校验的动作。
func (plan PrivacyPersistPlan) HasChanges() bool {
	return len(plan.FileUpdates) > 0 || len(plan.ForceFileUpdates) > 0 || len(plan.RuntimeEnvOnlyPaths) > 0
}
