# 部署说明

当前部署能力是生产风格示例，不是 v1 发布保证。真实环境使用前需要审查配置、密钥、数据库选择和回滚策略。

## 相关文件

| 路径 | 用途 |
| --- | --- |
| `Dockerfile` | 构建服务镜像 |
| `deploy/config.production.example.yaml` | 生产风格应用配置 |
| `deploy/docker-compose.production.example.yml` | Compose 服务定义 |
| `deploy.sh` | Bash 部署包装脚本 |
| `script/install.sh` | 远程安装入口，克隆仓库后委托仓库内 `deploy.sh` |
| `.github/workflows/deploy-remote.yml` | 手动触发的 GitHub Actions 远程部署 |

## 手动 Docker Compose 路径

```bash
mkdir -p /opt/go-scaffold/configs /var/lib/go-scaffold /var/log/go-scaffold
cp deploy/config.production.example.yaml /opt/go-scaffold/configs/config.yaml

export DEPLOY_IMAGE=go-scaffold:local
export APP_CONTAINER_PORT=9999
export HOST_CONFIG_FILE=/opt/go-scaffold/configs/config.yaml
export HOST_DATA_DIR=/var/lib/go-scaffold
export HOST_LOGS_DIR=/var/log/go-scaffold
export RIN_APP_AUTH_SIGNING_KEY=change-me-at-least-32-bytes-long
export RIN_APP_AUTH_REFRESH_TOKEN_PEPPER=change-me-refresh-pepper
export RIN_APP_AUTH_MFA_SECRET_KEY=change-me-mfa-secret-key-32-bytes
docker compose -f deploy/docker-compose.production.example.yml up -d
```

Compose 模板不再隐式绑定仓库相对目录；`HOST_CONFIG_FILE`、`HOST_DATA_DIR` 和
`HOST_LOGS_DIR` 必须显式指向宿主机路径。`deploy.sh` 会自动创建这些路径并导出给
Compose，手动运行 Compose 时需要自行准备。

然后检查：

```bash
curl http://127.0.0.1:9999/health
curl http://127.0.0.1:9999/ready
curl http://127.0.0.1:9999/admin
```

## deploy.sh 路径

`deploy.sh` 可以克隆仓库或使用本地仓库、准备配置、构建或拉取镜像、运行 Compose，并检查健康、就绪和 React WebUI 静态路由。破坏性或类生产操作必须显式传入 `--confirm`。

该脚本应在 Linux Bash 环境运行。

## GitHub 代理远程一键部署

`home-gh.helloworlds.eu.org` 给出的代理规则是把 GitHub 原始域名里的点替换为短横线，
并在末尾追加 `-gh.helloworlds.eu.org`。因此 raw 文件下载和 Git 克隆要分别处理：

| 原始用途 | 原始域名 | 代理域名 |
| --- | --- | --- |
| raw 脚本下载 | `raw.githubusercontent.com` | `raw-githubusercontent-com-gh.helloworlds.eu.org` |
| Git 克隆 | `github.com` | `github-com-gh.helloworlds.eu.org` |

远程 raw 文件只会执行 GitHub 上已经发布的脚本版本。仓库名必须由调用方显式传入；
raw URL 只负责下载入口脚本，后续 Git 克隆源由 `--repo` 或
`--github-proxy-host` + `--repo-slug` 决定。如果远程 `main` 的
`script/install.sh` 还只支持 `--repo` 和 `--ref`，使用下面这个兼容命令：

```bash
curl -fsSL https://raw-githubusercontent-com-gh.helloworlds.eu.org/rin721/aoi-server/main/script/install.sh | bash -s -- \
  --repo https://github-com-gh.helloworlds.eu.org/rin721/aoi-server.git \
  --docker y \
  --path /opt/go-scaffold \
  --image go-scaffold:local \
  --build y \
  --auth-signing-key change-me-at-least-32-bytes-long \
  --auth-refresh-token-pepper change-me-refresh-pepper \
  --auth-mfa-secret-key change-me-mfa-secret-key-32-bytes \
  --webui-mount-path / \
  --webui-check-path /admin \
  --confirm
```

如果仓库不是 `rin721/aoi-server`，把命令里的两处 `rin721/aoi-server` 改成实际的
`<owner>/<repo>`；脚本不会内置仓库名。raw 地址必须包含完整 `<owner>/<repo>`，
不能只写仓库名。

这个兼容命令会把配置、数据和日志放在 `--path` 下的默认目录：
`/opt/go-scaffold/configs`、`/opt/go-scaffold/data`、`/opt/go-scaffold/logs`。

脚本更新发布后，可以改用更明确的 `--github-proxy-host` 和宿主机目录映射参数：

```bash
curl -fsSL https://raw-githubusercontent-com-gh.helloworlds.eu.org/rin721/aoi-server/main/script/install.sh | bash -s -- \
  --github-proxy-host github-com-gh.helloworlds.eu.org \
  --repo-slug rin721/aoi-server \
  --docker y \
  --path /opt/go-scaffold \
  --config-dir /opt/go-scaffold/configs \
  --data-dir /var/lib/go-scaffold \
  --logs-dir /var/log/go-scaffold \
  --image go-scaffold:local \
  --build y \
  --auth-signing-key change-me-at-least-32-bytes-long \
  --auth-refresh-token-pepper change-me-refresh-pepper \
  --auth-mfa-secret-key change-me-mfa-secret-key-32-bytes \
  --webui-mount-path / \
  --webui-check-path /admin \
  --confirm
```

在新版 `script/install.sh` 入口中，`--github-proxy-host` 只影响 bootstrap 阶段自动
生成的 Git 克隆地址，必须配合显式的 `--repo-slug <owner>/<repo>` 使用，且不会继续
传给仓库内的 `deploy.sh`；如果已经传入 `--repo`，则以 `--repo` 为准。raw 代理
只解决入口脚本下载，不能替代后续 Git 克隆代理。
若目标机器访问该 raw 代理时仍被重定向到 `raw.githubusercontent.com` 且无法连通，
需要换成可用的 raw 代理地址，或先把 `script/install.sh` 上传到目标机器再运行。

脚本参数优先级是命令行参数高于环境变量。基础部署参数可以先用环境变量给默认
值，再用 flag 局部覆盖，例如 `DEPLOY_PATH`、`DEPLOY_IMAGE`、`APP_PORT`、
`HOST_DATA_DIR`。应用运行期配置优先使用 `RIN_APP_*`，也可以通过脚本 flag 显式
设置，脚本会把这些值传给 Compose。

常用 Docker 和宿主机目录参数：

| 参数 | 说明 |
| --- | --- |
| `--path /opt/go-scaffold` | 远端运行目录，保存 Compose 文件；必须是非根绝对路径，默认 `/opt/go-scaffold`。 |
| `--config-dir /opt/go-scaffold/configs` | 宿主机配置目录，脚本会在缺少 `config.yaml` 时写入生产配置示例；必须是非根绝对路径。 |
| `--data-dir /var/lib/go-scaffold` | 宿主机数据目录，映射到容器 `/app/data`；必须是非根绝对路径。 |
| `--logs-dir /var/log/go-scaffold` | 宿主机日志目录，映射到容器 `/app/logs`；必须是非根绝对路径。 |
| `--image go-scaffold:local` | 构建或拉取后运行的镜像。 |
| `--repo https://github-com-gh.helloworlds.eu.org/<owner>/<repo>.git` | 显式 Git 克隆地址；传入后优先于 `--repo-slug`。 |
| `--repo-slug <owner>/<repo>` | 显式 GitHub 仓库 slug；配合 `--github-proxy-host` 生成 Git 克隆代理地址。 |
| `--github-proxy-host github-com-gh.helloworlds.eu.org` | Git 克隆代理主机；不会内置仓库名，必须配合 `--repo-slug` 或直接传 `--repo`。 |
| `--port 9999` | 宿主机 HTTP 端口，映射到容器内应用端口。 |
| `--server-port 9999` | 容器内 HTTP 端口，同时写入 `RIN_APP_SERVER_PORT`；默认 `9999`。 |
| `--build y` / `--pull y` | 二选一；本机源码构建或从镜像仓库拉取。 |

常用 WebUI 参数：

| 参数 | 说明 |
| --- | --- |
| `--webui-mount-path /` | Go 静态托管挂载路径，运行时写入 `RIN_APP_WEBUI_MOUNT_PATH`；默认从根路径托管统一 React SPA。 |
| `--webui-public-base-url /` | WebUI 公开基础路径，运行时写入 `RIN_APP_WEBUI_PUBLIC_BASE_URL`。 |
| `--webui-api-base-url ""` | React `VITE_PUBLIC_API_BASE_URL`；空值表示同源调用。 |
| `--webui-check y` | 部署后是否检查 WebUI 静态路由。 |
| `--webui-check-path /admin` | WebUI 静态路由检查路径；默认可检查 `/`、`/setup` 或 `/admin`。 |

示例：

```bash
bash deploy.sh \
  --docker y \
  --path /opt/go-scaffold \
  --config-dir /opt/go-scaffold/configs \
  --data-dir /var/lib/go-scaffold \
  --logs-dir /var/log/go-scaffold \
  --image go-scaffold:local \
  --build y \
  --auth-signing-key "$RIN_APP_AUTH_SIGNING_KEY" \
  --auth-refresh-token-pepper "$RIN_APP_AUTH_REFRESH_TOKEN_PEPPER" \
  --auth-mfa-secret-key "$RIN_APP_AUTH_MFA_SECRET_KEY" \
  --webui-mount-path / \
  --webui-check-path /admin \
  --confirm
```

密钥类值可以使用上面的显式参数，也可以提前导出 `RIN_APP_AUTH_SIGNING_KEY`、
`RIN_APP_AUTH_REFRESH_TOKEN_PEPPER`、`RIN_APP_AUTH_MFA_SECRET_KEY` 等环境变量。
生产环境更推荐由 CI/CD secrets、主机密钥管理或受控 shell 会话注入，避免命令行
历史和进程列表暴露敏感值。

远程执行时，GitHub Actions 会把当前仓库的 `script/install.sh` 通过 SSH 传入远端 Bash；该入口会克隆目标 ref，再委托仓库内 `deploy.sh` 执行真实部署。`DEPLOY_REPO` 必须在 GitHub Environment 或 Repository Variables 中显式配置为完整 Git URL，workflow 不会提供默认仓库。直接在目标机器运行时，也可以下载 `deploy.sh` 后让它自行克隆仓库。

## 发布清单

1. 选择并验证生产数据库驱动。
2. 注入 `AUTH_SIGNING_KEY`、`AUTH_REFRESH_TOKEN_PEPPER`、`AUTH_MFA_SECRET_KEY` 等敏感值。
3. 运行 `db migrate status` 并在维护窗口执行 `db migrate up`。
4. 通过 `iam bootstrap-admin --password-stdin` 创建初始管理员。
6. 审查 CORS origins 和 headers，确保需要浏览器调用 IAM 时允许 `Authorization`。
7. 验证 `/health`、`/ready`、`/`、`/setup` 和 `/admin`。
8. 运行根模块测试。
9. 在干净环境构建 Docker 镜像。
10. 记录回滚、备份和迁移证据。
11. 如果属于托管任务，在对应运行时制品中记录部署证据。
## React WebUI 发布说明

生产镜像会从 `web/app` 构建统一 React WebUI 静态产物，并在运行时由 Go 服务从 `/` 托管。生产配置应保持：

```yaml
webui:
  enabled: true
  mount_path: /
  dist_dir: ./web/app/build/client
  public_base_url: ${RIN_APP_WEBUI_PUBLIC_BASE_URL:/}
```

手动发布非 Docker 产物时，需要先在 `web/app` 执行 `pnpm build`，再将 `web/app/build/client` 随服务一起部署。公开官网、首次安装向导和 `/admin` 后台共享同一套 React 构建产物。

Go 静态托管会保留 `/api`、`/api/v1`、`/health`、`/ready` 和插件协议路径，不会把这些路径回退到前端 `index.html`。修改 `webui.mount_path`、`webui.dist_dir` 或 `VITE_PUBLIC_API_BASE_URL` 后必须重新构建前端静态产物或 Docker 镜像。
