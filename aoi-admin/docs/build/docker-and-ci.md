# Docker 与 CI

构建、发布包和 Docker 镜像是三条不同路径，但它们必须使用同一套当前前端产物：`web/app/build/client`。

## 本地构建

```bash
go build -mod=readonly -o ./tmp/aoi-server ./cmd/aoi
pnpm --dir web/app build
```

Go 构建只生成服务二进制。React WebUI 构建会运行 Markdown 内容生成、React Router build，并校验 `web/app/build/client/index.html` 存在。

## 发布包

发布包统一使用 `scripts/package.py`；PowerShell 和 Bash 入口只负责透传参数：

```bash
python scripts/package.py
pwsh scripts/package.ps1
bash scripts/package.sh
```

常用示例：

```bash
python scripts/package.py --target linux/amd64 --target windows/amd64
python scripts/package.py --output build/releases --skip-web-build
python scripts/package.py --cgo
python scripts/package.py --dry-run
```

每个发布包包含：

- `aoi-server` 或 `aoi-server.exe`
- `configs/config.yaml`，来源于 `deploy/config.production.example.yaml`
- `configs/config.example.yaml` 和 `configs/locales/`
- `internal/migrations/`
- 可选的 `web/app/build/client/`
- 空的 `data/`、`logs/`
- `README.txt`
- `manifest.json`

参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--target` | 三平台 amd64 | 可重复传入，格式为 `goos/goarch`，也支持逗号分隔 |
| `--output` | `build/releases` | 发布包输出目录 |
| `--cgo` | `false` | 使用 `CGO_ENABLED=1` 构建 |
| `--skip-web-build` | `false` | 跳过 React WebUI build；若已有静态产物则打入包 |
| `--webui-build-base-url` | `/` | 记录到发布清单中的 WebUI 公开基础路径 |
| `--webui-api-base-url` | 空字符串 | 传给 React 构建的 `VITE_PUBLIC_API_BASE_URL`；空值表示同源调用 |
| `--clean` | `false` | 打包前清空输出目录 |
| `--dry-run` | `false` | 只打印计划，不写发布包 |
| `--verbose` | `false` | 输出外部命令细节 |
| `--version` | `AOI_VERSION` 或当前 git commit | 覆盖归档和 manifest 版本 |

默认发布包使用 `CGO_ENABLED=0`，SQLite 运行时不可用；需要 SQLite 时在目标平台使用 `--cgo` 重建，或改用 MySQL/PostgreSQL。

## Docker

```bash
docker build -t aoi-admin:local .
```

Dockerfile 包含 `web-build` 阶段：在 `web/app` 执行 `pnpm build`，检查 `build/client/index.html`，并把产物复制到运行镜像：

```text
/app/web/app/build/client
```

当前 WebUI Docker build arg：

| Build arg | 默认值 | 用途 |
| --- | --- | --- |
| `VITE_PUBLIC_API_BASE_URL` | 空字符串 | React API base URL；空值表示同源调用 Go API |

示例：

```bash
docker build \
  --build-arg VITE_PUBLIC_API_BASE_URL= \
  -t aoi-admin:local .
```

生产配置默认：

```yaml
webui:
  enabled: true
  mount_path: /
  dist_dir: ./web/app/build/client
  public_base_url: /
```

Go 静态托管从根路径 `/` 服务统一 SPA，并显式保留 `/api`、`/api/v1`、`/health`、`/ready` 和插件协议路径不进入前端 fallback。

## CI

`.github/workflows/ci.yml` 执行：

- 根据 `go.mod` 设置 Go
- 根据 `web/app/pnpm-lock.yaml` 设置 Node 和 pnpm 缓存
- 安装 React WebUI 依赖
- 报告 gofmt drift
- `go test ./... -count=1 -mod=readonly`
- server 构建
- React WebUI `lint:i18n`
- React WebUI `lint`
- React WebUI `typecheck`
- React WebUI `test:unit`
- React WebUI `build`
- Docker 镜像构建
- 空白检查

不要把构建期 secret 写入 Docker 镜像。运行期 secret 必须通过环境变量或密钥管理注入。
