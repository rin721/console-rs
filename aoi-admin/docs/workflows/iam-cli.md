# IAM CLI 工作流

`iam` 命令用于执行 IAM 初始化和运维入口。命令声明位于 `cmd/aoi`，实际组织、账号、角色、权限和审计行为由 `internal/modules/iam` service 处理。

## 首次初始化

本地默认 `migration.auto_apply=true`，server 和 IAM CLI 会在启动装配阶段自动应用 goose 迁移。需要手动检查或生产发布时，先显式应用迁移：

```bash
go run ./cmd/aoi db migrate up --config=configs/config.yaml
```

然后创建初始平台组织 owner：

```bash
go run ./cmd/aoi iam bootstrap-admin \
  --config=configs/config.yaml \
  --org-code=acme \
  --org-name="Acme Corp" \
  --username=admin \
  --email=admin@example.com \
  --password-stdin
```

建议在自动化脚本中通过标准输入或 secrets 管道传入密码，避免把密码写入 shell history。

## bootstrap-admin 行为

`iam bootstrap-admin` 会创建或复用目标平台组织和管理员用户，并确保：

- 管理员属于目标组织；
- 内置权限已写入 `iam_permissions`；
- `platform_owner`、`owner`、`admin`、`member` 系统角色已写入平台组织；
- `platform_owner` 的平台 Casbin policy 已写入 `iam_casbin_rules`；
- 管理员拥有目标平台组织的 `platform_owner` 角色；
- 审计日志记录初始化动作。

重复执行同一组织和同一管理员时应尽量保持幂等；如果用户名、邮箱或组织 code 与已有数据冲突，数据库唯一约束仍会保护数据一致性。

## 配置要求

IAM CLI 使用和 server 相同的配置加载路径：

- `--config` 优先；
- 未传入时读取 `RIN_CONFIG_PATH`；
- 再回退到 `configs/config.yaml`。

生产环境必须注入：

- `AUTH_SIGNING_KEY` 或 `RIN_APP_AUTH_SIGNING_KEY`；
- `AUTH_REFRESH_TOKEN_PEPPER` 或 `RIN_APP_AUTH_REFRESH_TOKEN_PEPPER`；
- `AUTH_MFA_SECRET_KEY` 或 `RIN_APP_AUTH_MFA_SECRET_KEY`。

## 后续运维

用户邀请、角色、权限、会话撤销和审计查看由 HTTP 管理接口提供；CLI 不重复实现这些后台管理页面。需要本地一站式初始化时，使用 System Center `init`：

```bash
go run ./cmd/aoi init \
  --config=configs/config.yaml \
  --admin-username=admin \
  --admin-email=admin@example.com \
  --admin-password-stdin \
  --create-service-token
```

`init` 通过 `internal/app/initcenter` 统一编排迁移、System 默认数据、API/权限同步、管理员创建、可选服务 API Token 和最终验证，并记录运行/步骤状态。受管服务入口位于同一组命令：

```bash
go build -mod=readonly -o ./tmp/go-scaffold-server.exe ./cmd/aoi
./tmp/go-scaffold-server.exe run server --config=configs/config.yaml
./tmp/go-scaffold-server.exe service status server
./tmp/go-scaffold-server.exe service info server
./tmp/go-scaffold-server.exe service logs server
./tmp/go-scaffold-server.exe service terminal server
./tmp/go-scaffold-server.exe service restart server
./tmp/go-scaffold-server.exe service stop server
```

同一套菜单 prompt 也可通过 `--chain.*` 自动回答，适合脚本里跳过后续动态选择：

```powershell
.\tmp\go-scaffold-server.exe run --chain.service=server --chain.config=configs/config.yaml --chain.privacy=false
.\tmp\go-scaffold-server.exe service --chain.action=status
.\tmp\go-scaffold-server.exe service --chain.action=logs --chain.logs.follow=true
python scripts/package.py --target windows/amd64 --output build/releases --skip-web-generate
```

隐私配置后续输入也在同一套 key 下，例如截图里的 `auth.signing_key` 输入可以用 `--chain.privacy.auth.signing_key.value=generate` 自动回答；路径较多时可用 `--chain.privacy.auth.*.value=generate` 这类通配 key。

`init` 的链式 key 与现有 flag 语义一致，例如 `--chain.org-code`、`--chain.admin-password`、`--chain.create-service-token`、`--chain.service-token-days`。普通未知 flag 仍会报错，只有 `--chain.<key>` 会作为 prompt 答案进入后续流程。

System Center 默认把运行态记录放在 `data/cli-runtime`，可通过 `RIN_CLI_RUNTIME_DIR` 覆盖。受管进程会设置 `RIN_CLI_MANAGED` 和 `RIN_CLI_SERVICE`，用于区分手动启动和 CLI 托管启动。
前台调试可继续使用 `go run ./cmd/aoi server --config=configs/config.yaml`。后台托管优先使用固定二进制；Windows 下 `go run ./cmd/aoi run server` 会先落到 Go 临时目录，服务常驻时可能锁住 `go-build...\main.exe` 并导致父进程清理时报 `Access is denied`。
