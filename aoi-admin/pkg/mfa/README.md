# pkg/mfa - 多因素认证封装

`pkg/mfa` 是项目的 TOTP 防腐层。它把 `github.com/pquerna/otp` 收敛在 `pkg` 内部，向业务层暴露小而稳定的 TOTP API。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`TOTPKey`、`GenerateTOTP`、`ValidateTOTP`、`GenerateTOTPCode`。
- 当前风险：[RISK] 当前只支持 TOTP，不支持短信、邮件验证码、WebAuthn 或恢复码。
- 非目标：[CONFIRMED] 本包不保存 MFA secret、不负责加密、不管理用户 MFA 状态。

## 基本用法

```go
key, err := mfa.GenerateTOTP("go-scaffold", "admin@example.com")
if err != nil {
    return err
}

// key.Secret 应由调用方加密后保存。
// key.URL 可展示为二维码或直接交给认证器 App。

if !mfa.ValidateTOTP(code, key.Secret) {
    return ErrUnauthorized
}
```

## IAM 集成

IAM service 使用本包生成 TOTP secret 和 otpauth URL。secret 的加密、确认状态、启用状态和登录校验策略留在 `internal/modules/iam` 中处理。
