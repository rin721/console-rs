package mfa

import (
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

// TOTPKey 是创建 TOTP 因子后返回给调用方的密钥材料；Secret 需持久化，URL 用于二维码绑定。
type TOTPKey struct {
	Secret string
	URL    string
}

// GenerateTOTP 为指定发行方和账号生成 TOTP 密钥，调用方负责将 Secret 安全保存到用户 MFA 配置中。
func GenerateTOTP(issuer, accountName string) (TOTPKey, error) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: issuer, AccountName: accountName})
	if err != nil {
		return TOTPKey{}, err
	}
	return TOTPKey{Secret: key.Secret(), URL: key.URL()}, nil
}

// ValidateTOTP 校验用户输入的一次性验证码；输入会先去除首尾空白以兼容复制粘贴场景。
func ValidateTOTP(code, secret string) bool {
	return totp.Validate(strings.TrimSpace(code), secret)
}

// GenerateTOTPCode 在指定时间点生成验证码，主要用于测试和需要可重复校验的离线流程。
func GenerateTOTPCode(secret string, at time.Time) (string, error) {
	return totp.GenerateCode(secret, at)
}
