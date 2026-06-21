# pkg/token - Token 管理封装

`pkg/token` 是项目的 JWT 与 refresh token 哈希防腐层。它把 `github.com/golang-jwt/jwt/v5` 收敛在 `pkg` 内部，向业务层暴露项目自有的 `Manager`、`Subject`、`Claims` 和 `Pair`。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`Manager` 接口、`Config`、`Subject`、`Claims`、`Pair`、token 类型常量和错误值。
- 当前风险：[RISK] 当前实现使用 HS256，对非对称签名、密钥轮换和多 issuer 策略只保留后续扩展空间。
- 非目标：[CONFIRMED] 本包不访问数据库、不判断用户状态、不管理会话撤销，也不负责 HTTP Bearer 解析。

## 设计约束

- access token 使用 JWT，claims 固定包含 `userId`、`orgId`、`sessionId`、`tokenType`。
- refresh token 是随机字符串，数据库只应保存 `HashRefreshToken` 生成的 HMAC/SHA-256 hash。
- `Parse` 会校验 issuer、audience、签名、过期时间和 `tokenType`。
- 业务层不应直接导入 JWT 库。

## 基本用法

```go
manager, err := token.New(token.Config{
    Issuer:        "go-scaffold",
    Audience:      []string{"go-scaffold-api"},
    SigningKey:    "change-me-at-least-32-bytes-long",
    AccessTTL:     15 * time.Minute,
    RefreshTTL:    7 * 24 * time.Hour,
    RefreshPepper: "change-me-refresh-pepper",
})
if err != nil {
    return err
}

pair, err := manager.IssuePair(ctx, token.Subject{
    UserID:    1001,
    OrgID:     2001,
    SessionID: 3001,
})
if err != nil {
    return err
}

claims, err := manager.Parse(ctx, pair.AccessToken, token.TokenTypeAccess)
```

## IAM 集成

`internal/modules/iam` 使用本包完成登录、刷新、切换组织和 access token 认证。会话是否撤销、用户是否禁用、组织成员关系是否有效，由 IAM service 和 repository 基于数据库状态判断。
