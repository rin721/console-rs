# AI 与工具说明

本文记录当前 Codex 协作常用工具。安装状态会随机器变化；执行前以本机命令输出为准。

## 本地验证命令

```powershell
go test ./internal/config -count=1 -mod=readonly
go test ./internal/... -count=1 -mod=readonly
go test ./... -count=1 -mod=readonly
go vet ./...
go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi
git diff --check
```

架构边界扫描：

```powershell
rg -n "github\\.com/rei0721/go-scaffold/pkg/" internal/modules internal/middleware internal/transport --glob "*.go" --glob "!**/*_test.go"
rg -n "http\\.Client|smtp\\.|os\\.Getenv|database\\.New|WithExecutor|internal/modules/.*/repository" internal/modules/*/service --glob "*.go"
```

安全和质量工具：

```powershell
golangci-lint run --config tools/ai/golangci.yml ./...
govulncheck ./...
gosec ./...
osv-scanner scan source .
```

## GitHub 工作流

GitHub CLI 用于本地分支 PR 发现和 Actions 日志：

```powershell
gh auth status
gh pr view
gh pr checks
gh run view
```

如果 `gh auth status` 未登录，先请用户完成 `gh auth login`。PR、issue、评论和发布 PR 优先使用 GitHub connector 或对应 GitHub skill。

## Browser 工作流

涉及 `web/app` 页面、静态托管、响应式布局或可见交互时，需要用 Browser 检查本地页面。至少覆盖：

- 桌面：`1440x900`
- 移动：`390x844`
- 登录页和受影响业务页
- 控制台错误、横向溢出、`undefined/null/NaN` 文本

纯后端文档、配置或 Go-only 重构不需要打开浏览器。

## 工具边界

- 不为普通后端任务默认安装 Docker Desktop、Semgrep 或 Testcontainers。
- 不把临时扫描结果写进 `cmd`、`internal`、`pkg` 或 `types`。
- 长任务证据先放 `tmp/ai`；只有确认会长期维护的事实才进入 `docs/ai`。
