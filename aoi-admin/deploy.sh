#!/usr/bin/env bash
set -euo pipefail

DEFAULT_REPO_REF="main"

log() {
	printf '[deploy] %s\n' "$*"
}

die() {
	printf '[deploy] ERROR: %s\n' "$*" >&2
	exit 1
}

usage() {
	cat <<'USAGE'
用法:
  bash deploy.sh --docker y --confirm [options]

先克隆仓库:
  git clone <repo-address>
  cd <repo-address>
  bash deploy.sh --docker y --image go-scaffold:local --confirm

直接安装:
  curl -fsSL -o deploy.sh https://raw.githubusercontent.com/<owner>/<repo>/main/deploy.sh
  bash deploy.sh --repo https://github.com/<owner>/<repo>.git --docker y --image go-scaffold:local --confirm

GitHub 代理安装:
  curl -fsSL -o deploy.sh https://raw-githubusercontent-com-gh.helloworlds.eu.org/<owner>/<repo>/main/deploy.sh
  bash deploy.sh --github-proxy-host github-com-gh.helloworlds.eu.org --repo-slug <owner>/<repo> --docker y --image go-scaffold:local --confirm

部署选项:
  --docker <y|n>              使用 Docker Compose 部署；当前仅支持 "y"。
  --repo <url>                脚本不在仓库目录内运行时要克隆的仓库地址。
  --repo-slug <owner>/<repo>  GitHub 仓库 slug；配合 --github-proxy-host 生成克隆地址。
  --ref <ref>                 要克隆的 Git ref，默认 main。
  --github-proxy-host <host>  GitHub 克隆代理主机；必须配合 --repo-slug 或 --repo。
  --path <path>               运行目录，默认 /opt/go-scaffold。
  --config-dir <path>         宿主机配置目录，默认 <path>/configs。
  --data-dir <path>           宿主机数据目录，默认 <path>/data。
  --logs-dir <path>           宿主机日志目录，默认 <path>/logs。
  --image <image>             要构建或运行的镜像，默认 go-scaffold:local。
  --build <y|n>               是否从源码构建镜像，默认 y。
  --pull <y|n>                Compose up 前是否拉取镜像，默认 n。
  --port <port>               宿主机 HTTP 端口，默认 9999。
  --container-name <name>     Docker 容器名，默认 go-scaffold。
  --env <staging|production>  部署环境标签，默认 production。
  --confirm                   必需的确认标记。

可选环境变量:
  基础部署参数可用同名大写变量作为默认值，例如 DEPLOY_PATH、DEPLOY_IMAGE、
  APP_PORT、HOST_DATA_DIR。命令行参数始终优先于环境变量。

镜像仓库选项:
  --registry-host <host>      镜像仓库地址，默认 ghcr.io。
  --registry-username <name>  docker login 可选用户名。
  --registry-token <token>    docker login 可选令牌。

应用选项:
  --db-driver <value>
  --db-host <value>
  --db-port <value>
  --db-user <value>
  --db-password <value>
  --db-name <value>
  --db-max-open-conns <value>
  --db-max-idle-conns <value>
  --redis-enabled <value>
  --redis-host <value>
  --redis-port <value>
  --redis-password <value>
  --redis-db <value>
  --redis-pool-size <value>
  --redis-min-idle-conns <value>
  --redis-max-retries <value>
  --redis-dial-timeout <value>
  --redis-read-timeout <value>
  --redis-write-timeout <value>
  --server-host <value>
  --server-port <value>       容器内 HTTP 端口，同时写入 RIN_APP_SERVER_PORT，默认 9999。
  --server-mode <value>
  --server-read-timeout <value>
  --server-write-timeout <value>
  --server-idle-timeout <value>
  --rpc-enabled <value>
  --rpc-host <value>
  --rpc-port <value>
  --rpc-read-timeout <value>
  --rpc-write-timeout <value>
  --rpc-idle-timeout <value>
  --webui-enabled <value>
  --webui-mount-path <value>
  --webui-dist-dir <value>
  --webui-public-base-url <value>
  --webui-api-base-url <url-or-path> React VITE_PUBLIC_API_BASE_URL，默认同源。
  --webui-check <y|n>                部署后是否检查 WebUI 静态路由，默认 y。
  --webui-check-path <path>          部署后检查路径，默认 /admin。
  --log-level <value>
  --log-format <value>
  --log-console-format <value>
  --log-file-format <value>
  --log-output <value>
  --log-file-path <value>
  --log-max-size <value>
  --log-max-backups <value>
  --log-max-age <value>
  --i18n-default-locale <value>
  --i18n-fallback-locale <value>
  --i18n-supported-locales <value>
  --i18n-resources-ui <value>
  --i18n-resources-api <value>
  --i18n-resources-validation <value>
  --i18n-resources-system <value>
  --executor-enabled <value>
  --storage-driver <value>
  --storage-local-fs-type <value>
  --storage-local-base-path <value>
  --storage-local-enable-watch <value>
  --storage-local-watch-buffer-size <value>
  --system-seed-defaults-on-start <value>
  --plugins-enabled <value>
  --plugins-heartbeat-timeout-seconds <value>
  --auth-enabled <value>
  --auth-registration-mode <disabled|direct|email_verification|invite_only>
  --auth-issuer <value>
  --auth-audience <value>
  --auth-signing-key <value>
  --auth-access-token-ttl-seconds <value>
  --auth-refresh-token-ttl-seconds <value>
  --auth-refresh-token-pepper <value>
  --auth-mfa-issuer <value>
  --auth-mfa-secret-key <value>
  --auth-login-max-failures <value>
  --auth-login-lock-minutes <value>
  --auth-login-captcha-enabled <value>
  --auth-captcha-ttl-seconds <value>
  --auth-email-verification-ttl-seconds <value>
  --auth-invitation-ttl-seconds <value>
  --auth-password-reset-ttl-seconds <value>
  --auth-notification-driver <value>
  --auth-smtp-host <value>
  --auth-smtp-port <value>
  --auth-smtp-username <value>
  --auth-smtp-password <value>
  --auth-smtp-from <value>
  --auth-smtp-from-name <value>
  --auth-smtp-security <none|starttls|tls>
  --auth-password-min-length <value>
  --auth-password-require-lower <value>
  --auth-password-require-upper <value>
  --auth-password-require-number <value>
  --auth-password-require-symbol <value>
  --auth-casbin-reload-interval-seconds <value>
  --migration-auto-apply <value>
  --migration-dir <value>
  --cors-enabled <value>
  --cors-allow-origins <value>
  --cors-allow-methods <value>
  --cors-allow-headers <value>
  --cors-expose-headers <value>
  --cors-allow-credentials <value>
  --cors-max-age <value>

安全提示:
  密码、令牌和密钥类参数可能出现在 shell 历史或进程列表中。
  优先使用受控 shell 会话、CI secret masking 或主机密钥管理服务。
  本脚本不会主动打印敏感值。
USAGE
}

require_arg() {
	local flag="$1"
	local value="${2:-}"
	[ -n "$value" ] || die "$flag requires a value"
}

normalize_yn() {
	local value="${1,,}"
	case "$value" in
	y | yes | true | 1) printf 'y' ;;
	n | no | false | 0) printf 'n' ;;
	*) die "expected y or n, got: $1" ;;
	esac
}

normalize_bool_string() {
	local value="${1,,}"
	case "$value" in
	y | yes | true | 1) printf 'true' ;;
	n | no | false | 0) printf 'false' ;;
	*) die "expected boolean value, got: $1" ;;
	esac
}

normalize_webui_mount_path() {
	local value="$1"
	[ -n "$value" ] || die "webui mount path cannot be empty"
	[[ "$value" == /* ]] || die "webui mount path must start with /"
	if [ "$value" = "/" ]; then
		printf '/'
		return 0
	fi
	printf '%s' "${value%/}"
}

validate_value() {
	local key="$1"
	local value="$2"
	if [[ "$value" == *$'\n'* || "$value" == *$'\r'* ]]; then
		die "$key cannot contain newlines"
	fi
}

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

repo_url_from_github_proxy() {
	local host="$1"
	local slug="$2"

	host="${host#https://}"
	host="${host#http://}"
	host="${host%/}"
	slug="${slug#https://github.com/}"
	slug="${slug#http://github.com/}"
	slug="${slug%.git}"
	slug="${slug#/}"
	slug="${slug%/}"
	[ -n "$host" ] || die "github proxy host cannot be empty"
	[ -n "$slug" ] || die "--repo-slug is required with --github-proxy-host when --repo is not set"
	[[ "$slug" != *"://"* ]] || die "--repo-slug must be owner/name, not a URL"
	[[ "$slug" == */* ]] || die "--repo-slug must be owner/name"
	[[ "$slug" != */*/* ]] || die "--repo-slug must be owner/name"
	printf 'https://%s/%s.git' "$host" "$slug"
}

clone_repo() {
	local repo_url="$1"
	local repo_ref="$2"
	local target_dir="$3"

	require_cmd git
	log "cloning repository"
	if git clone --depth 1 --branch "$repo_ref" "$repo_url" "$target_dir" >/dev/null 2>&1; then
		return 0
	fi

	rm -rf "$target_dir"
	git clone "$repo_url" "$target_dir" >/dev/null
	git -C "$target_dir" checkout "$repo_ref" >/dev/null
}

DEPLOY_DOCKER="${DEPLOY_DOCKER:-}"
REPO_URL="${REPO_URL:-${DEPLOY_REPO_URL:-}}"
REPO_REF="${REPO_REF:-${DEPLOY_REPO_REF:-$DEFAULT_REPO_REF}}"
REPO_SLUG="${REPO_SLUG:-${DEPLOY_REPO_SLUG:-}}"
GITHUB_PROXY_HOST="${GITHUB_PROXY_HOST:-${DEPLOY_GITHUB_PROXY_HOST:-}}"
DEPLOY_PATH="${DEPLOY_PATH:-/opt/go-scaffold}"
HOST_CONFIG_DIR="${HOST_CONFIG_DIR:-}"
HOST_DATA_DIR="${HOST_DATA_DIR:-}"
HOST_LOGS_DIR="${HOST_LOGS_DIR:-}"
DEPLOY_IMAGE="${DEPLOY_IMAGE:-go-scaffold:local}"
DEPLOY_BUILD="${DEPLOY_BUILD:-y}"
DEPLOY_BUILD_SET="n"
DEPLOY_PULL="${DEPLOY_PULL:-n}"
APP_PORT="${APP_PORT:-9999}"
APP_CONTAINER_PORT="${APP_CONTAINER_PORT:-${RIN_APP_SERVER_PORT:-9999}}"
DEPLOY_CONTAINER_NAME="${DEPLOY_CONTAINER_NAME:-go-scaffold}"
DEPLOY_ENV="${DEPLOY_ENV:-production}"
DEPLOY_CONFIRM="n"
SOURCE_DIR="${SOURCE_DIR:-}"
REGISTRY_HOST="${REGISTRY_HOST:-ghcr.io}"
REGISTRY_USERNAME="${REGISTRY_USERNAME:-}"
REGISTRY_TOKEN="${REGISTRY_TOKEN:-}"
APP_ENV_PREFIX="${APP_ENV_PREFIX:-RIN_APP}"
WEBUI_API_BASE_URL="${WEBUI_API_BASE_URL:-}"
WEBUI_CHECK="${WEBUI_CHECK:-y}"
WEBUI_CHECK_SET="n"
WEBUI_CHECK_PATH="${WEBUI_CHECK_PATH:-}"

declare -A APP_ENV=()

set_app_env() {
	local key="$1"
	local value="$2"
	validate_value "$key" "$value"
	APP_ENV["${APP_ENV_PREFIX}_${key}"]="$value"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--docker)
		require_arg "$1" "${2:-}"
		DEPLOY_DOCKER="$(normalize_yn "$2")"
		shift 2
		;;
	--repo)
		require_arg "$1" "${2:-}"
		REPO_URL="$2"
		shift 2
		;;
	--ref)
		require_arg "$1" "${2:-}"
		REPO_REF="$2"
		shift 2
		;;
	--repo-slug)
		require_arg "$1" "${2:-}"
		REPO_SLUG="$2"
		shift 2
		;;
	--github-proxy-host)
		require_arg "$1" "${2:-}"
		GITHUB_PROXY_HOST="$2"
		shift 2
		;;
	--path)
		require_arg "$1" "${2:-}"
		DEPLOY_PATH="$2"
		shift 2
		;;
	--config-dir)
		require_arg "$1" "${2:-}"
		HOST_CONFIG_DIR="$2"
		shift 2
		;;
	--data-dir)
		require_arg "$1" "${2:-}"
		HOST_DATA_DIR="$2"
		shift 2
		;;
	--logs-dir)
		require_arg "$1" "${2:-}"
		HOST_LOGS_DIR="$2"
		shift 2
		;;
	--image)
		require_arg "$1" "${2:-}"
		DEPLOY_IMAGE="$2"
		shift 2
		;;
	--build)
		require_arg "$1" "${2:-}"
		DEPLOY_BUILD="$(normalize_yn "$2")"
		DEPLOY_BUILD_SET="y"
		shift 2
		;;
	--pull)
		require_arg "$1" "${2:-}"
		DEPLOY_PULL="$(normalize_yn "$2")"
		shift 2
		;;
	--port)
		require_arg "$1" "${2:-}"
		APP_PORT="$2"
		shift 2
		;;
	--container-name)
		require_arg "$1" "${2:-}"
		DEPLOY_CONTAINER_NAME="$2"
		shift 2
		;;
	--env)
		require_arg "$1" "${2:-}"
		DEPLOY_ENV="$2"
		shift 2
		;;
	--confirm)
		DEPLOY_CONFIRM="y"
		shift
		;;
	--source-dir)
		require_arg "$1" "${2:-}"
		SOURCE_DIR="$2"
		shift 2
		;;
	--registry-host)
		require_arg "$1" "${2:-}"
		REGISTRY_HOST="$2"
		shift 2
		;;
	--registry-username)
		require_arg "$1" "${2:-}"
		REGISTRY_USERNAME="$2"
		shift 2
		;;
	--registry-token)
		require_arg "$1" "${2:-}"
		REGISTRY_TOKEN="$2"
		shift 2
		;;
	--db-driver)
		require_arg "$1" "${2:-}"
		set_app_env DB_DRIVER "$2"
		shift 2
		;;
	--db-host)
		require_arg "$1" "${2:-}"
		set_app_env DB_HOST "$2"
		shift 2
		;;
	--db-port)
		require_arg "$1" "${2:-}"
		set_app_env DB_PORT "$2"
		shift 2
		;;
	--db-user)
		require_arg "$1" "${2:-}"
		set_app_env DB_USER "$2"
		shift 2
		;;
	--db-password)
		require_arg "$1" "${2:-}"
		set_app_env DB_PASSWORD "$2"
		shift 2
		;;
	--db-name)
		require_arg "$1" "${2:-}"
		set_app_env DB_NAME "$2"
		shift 2
		;;
	--db-max-open-conns)
		require_arg "$1" "${2:-}"
		set_app_env DB_MAX_OPEN_CONNS "$2"
		shift 2
		;;
	--db-max-idle-conns)
		require_arg "$1" "${2:-}"
		set_app_env DB_MAX_IDLE_CONNS "$2"
		shift 2
		;;
	--redis-enabled)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_ENABLED "$2"
		shift 2
		;;
	--redis-host)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_HOST "$2"
		shift 2
		;;
	--redis-port)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_PORT "$2"
		shift 2
		;;
	--redis-password)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_PASSWORD "$2"
		shift 2
		;;
	--redis-db)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_DB "$2"
		shift 2
		;;
	--redis-pool-size)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_POOL_SIZE "$2"
		shift 2
		;;
	--redis-min-idle-conns)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_MIN_IDLE_CONNS "$2"
		shift 2
		;;
	--redis-max-retries)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_MAX_RETRIES "$2"
		shift 2
		;;
	--redis-dial-timeout)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_DIAL_TIMEOUT "$2"
		shift 2
		;;
	--redis-read-timeout)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_READ_TIMEOUT "$2"
		shift 2
		;;
	--redis-write-timeout)
		require_arg "$1" "${2:-}"
		set_app_env REDIS_WRITE_TIMEOUT "$2"
		shift 2
		;;
	--server-host)
		require_arg "$1" "${2:-}"
		set_app_env SERVER_HOST "$2"
		shift 2
		;;
	--server-port)
		require_arg "$1" "${2:-}"
		APP_CONTAINER_PORT="$2"
		set_app_env SERVER_PORT "$2"
		shift 2
		;;
	--server-mode)
		require_arg "$1" "${2:-}"
		set_app_env SERVER_MODE "$2"
		shift 2
		;;
	--server-read-timeout)
		require_arg "$1" "${2:-}"
		set_app_env SERVER_READ_TIMEOUT "$2"
		shift 2
		;;
	--server-write-timeout)
		require_arg "$1" "${2:-}"
		set_app_env SERVER_WRITE_TIMEOUT "$2"
		shift 2
		;;
	--server-idle-timeout)
		require_arg "$1" "${2:-}"
		set_app_env SERVER_IDLE_TIMEOUT "$2"
		shift 2
		;;
	--rpc-enabled)
		require_arg "$1" "${2:-}"
		set_app_env RPC_ENABLED "$2"
		shift 2
		;;
	--rpc-host)
		require_arg "$1" "${2:-}"
		set_app_env RPC_HOST "$2"
		shift 2
		;;
	--rpc-port)
		require_arg "$1" "${2:-}"
		set_app_env RPC_PORT "$2"
		shift 2
		;;
	--rpc-read-timeout)
		require_arg "$1" "${2:-}"
		set_app_env RPC_READ_TIMEOUT "$2"
		shift 2
		;;
	--rpc-write-timeout)
		require_arg "$1" "${2:-}"
		set_app_env RPC_WRITE_TIMEOUT "$2"
		shift 2
		;;
	--rpc-idle-timeout)
		require_arg "$1" "${2:-}"
		set_app_env RPC_IDLE_TIMEOUT "$2"
		shift 2
		;;
	--webui-enabled)
		require_arg "$1" "${2:-}"
		set_app_env WEBUI_ENABLED "$2"
		shift 2
		;;
	--webui-mount-path)
		require_arg "$1" "${2:-}"
		set_app_env WEBUI_MOUNT_PATH "$(normalize_webui_mount_path "$2")"
		shift 2
		;;
	--webui-dist-dir)
		require_arg "$1" "${2:-}"
		set_app_env WEBUI_DIST_DIR "$2"
		shift 2
		;;
	--webui-public-base-url)
		require_arg "$1" "${2:-}"
		set_app_env WEBUI_PUBLIC_BASE_URL "$2"
		shift 2
		;;
	--webui-api-base-url)
		require_arg "$1" "${2:-}"
		WEBUI_API_BASE_URL="$2"
		shift 2
		;;
	--webui-check)
		require_arg "$1" "${2:-}"
		WEBUI_CHECK="$(normalize_yn "$2")"
		WEBUI_CHECK_SET="y"
		shift 2
		;;
	--webui-check-path)
		require_arg "$1" "${2:-}"
		WEBUI_CHECK_PATH="$2"
		shift 2
		;;
	--log-level)
		require_arg "$1" "${2:-}"
		set_app_env LOG_LEVEL "$2"
		shift 2
		;;
	--log-format)
		require_arg "$1" "${2:-}"
		set_app_env LOG_FORMAT "$2"
		shift 2
		;;
	--log-console-format)
		require_arg "$1" "${2:-}"
		set_app_env LOG_CONSOLE_FORMAT "$2"
		shift 2
		;;
	--log-file-format)
		require_arg "$1" "${2:-}"
		set_app_env LOG_FILE_FORMAT "$2"
		shift 2
		;;
	--log-output)
		require_arg "$1" "${2:-}"
		set_app_env LOG_OUTPUT "$2"
		shift 2
		;;
	--log-file-path)
		require_arg "$1" "${2:-}"
		set_app_env LOG_FILE_PATH "$2"
		shift 2
		;;
	--log-max-size)
		require_arg "$1" "${2:-}"
		set_app_env LOG_MAX_SIZE "$2"
		shift 2
		;;
	--log-max-backups)
		require_arg "$1" "${2:-}"
		set_app_env LOG_MAX_BACKUPS "$2"
		shift 2
		;;
	--log-max-age)
		require_arg "$1" "${2:-}"
		set_app_env LOG_MAX_AGE "$2"
		shift 2
		;;
	--i18n-default-locale)
		require_arg "$1" "${2:-}"
		set_app_env I18N_DEFAULT_LOCALE "$2"
		shift 2
		;;
	--i18n-fallback-locale)
		require_arg "$1" "${2:-}"
		set_app_env I18N_FALLBACK_LOCALE "$2"
		shift 2
		;;
	--i18n-supported-locales)
		require_arg "$1" "${2:-}"
		set_app_env I18N_SUPPORTED_LOCALES "$2"
		shift 2
		;;
	--i18n-resources-ui)
		require_arg "$1" "${2:-}"
		set_app_env I18N_RESOURCES_UI "$2"
		shift 2
		;;
	--i18n-resources-api)
		require_arg "$1" "${2:-}"
		set_app_env I18N_RESOURCES_API "$2"
		shift 2
		;;
	--i18n-resources-validation)
		require_arg "$1" "${2:-}"
		set_app_env I18N_RESOURCES_VALIDATION "$2"
		shift 2
		;;
	--i18n-resources-system)
		require_arg "$1" "${2:-}"
		set_app_env I18N_RESOURCES_SYSTEM "$2"
		shift 2
		;;
	--executor-enabled)
		require_arg "$1" "${2:-}"
		set_app_env EXECUTOR_ENABLED "$2"
		shift 2
		;;
	--storage-driver)
		require_arg "$1" "${2:-}"
		set_app_env STORAGE_DRIVER "$2"
		shift 2
		;;
	--storage-local-fs-type)
		require_arg "$1" "${2:-}"
		set_app_env STORAGE_LOCAL_FS_TYPE "$2"
		shift 2
		;;
	--storage-local-base-path)
		require_arg "$1" "${2:-}"
		set_app_env STORAGE_LOCAL_BASE_PATH "$2"
		shift 2
		;;
	--storage-local-enable-watch)
		require_arg "$1" "${2:-}"
		set_app_env STORAGE_LOCAL_ENABLE_WATCH "$2"
		shift 2
		;;
	--storage-local-watch-buffer-size)
		require_arg "$1" "${2:-}"
		set_app_env STORAGE_LOCAL_WATCH_BUFFER_SIZE "$2"
		shift 2
		;;
	--system-seed-defaults-on-start)
		require_arg "$1" "${2:-}"
		set_app_env SYSTEM_SEED_DEFAULTS_ON_START "$2"
		shift 2
		;;
	--plugins-enabled)
		require_arg "$1" "${2:-}"
		set_app_env PLUGINS_ENABLED "$2"
		shift 2
		;;
	--plugins-heartbeat-timeout-seconds)
		require_arg "$1" "${2:-}"
		set_app_env PLUGINS_HEARTBEAT_TIMEOUT_SECONDS "$2"
		shift 2
		;;
	--auth-enabled)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_ENABLED "$2"
		shift 2
		;;
	--auth-registration-mode)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_REGISTRATION_MODE "$2"
		shift 2
		;;
	--auth-issuer)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_ISSUER "$2"
		shift 2
		;;
	--auth-audience)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_AUDIENCE "$2"
		shift 2
		;;
	--auth-signing-key)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SIGNING_KEY "$2"
		shift 2
		;;
	--auth-access-token-ttl-seconds)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_ACCESS_TOKEN_TTL_SECONDS "$2"
		shift 2
		;;
	--auth-refresh-token-ttl-seconds)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_REFRESH_TOKEN_TTL_SECONDS "$2"
		shift 2
		;;
	--auth-refresh-token-pepper)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_REFRESH_TOKEN_PEPPER "$2"
		shift 2
		;;
	--auth-mfa-issuer)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_MFA_ISSUER "$2"
		shift 2
		;;
	--auth-mfa-secret-key)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_MFA_SECRET_KEY "$2"
		shift 2
		;;
	--auth-login-max-failures)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_LOGIN_MAX_FAILURES "$2"
		shift 2
		;;
	--auth-login-lock-minutes)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_LOGIN_LOCK_MINUTES "$2"
		shift 2
		;;
	--auth-login-captcha-enabled)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_LOGIN_CAPTCHA_ENABLED "$2"
		shift 2
		;;
	--auth-captcha-ttl-seconds)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_CAPTCHA_TTL_SECONDS "$2"
		shift 2
		;;
	--auth-email-verification-ttl-seconds)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_EMAIL_VERIFICATION_TTL_SECONDS "$2"
		shift 2
		;;
	--auth-invitation-ttl-seconds)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_INVITATION_TTL_SECONDS "$2"
		shift 2
		;;
	--auth-password-reset-ttl-seconds)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_PASSWORD_RESET_TTL_SECONDS "$2"
		shift 2
		;;
	--auth-notification-driver)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_NOTIFICATION_DRIVER "$2"
		shift 2
		;;
	--auth-smtp-host)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SMTP_HOST "$2"
		shift 2
		;;
	--auth-smtp-port)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SMTP_PORT "$2"
		shift 2
		;;
	--auth-smtp-username)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SMTP_USERNAME "$2"
		shift 2
		;;
	--auth-smtp-password)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SMTP_PASSWORD "$2"
		shift 2
		;;
	--auth-smtp-from)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SMTP_FROM "$2"
		shift 2
		;;
	--auth-smtp-from-name)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SMTP_FROM_NAME "$2"
		shift 2
		;;
	--auth-smtp-security)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_SMTP_SECURITY "$2"
		shift 2
		;;
	--auth-password-min-length)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_PASSWORD_MIN_LENGTH "$2"
		shift 2
		;;
	--auth-password-require-lower)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_PASSWORD_REQUIRE_LOWER "$2"
		shift 2
		;;
	--auth-password-require-upper)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_PASSWORD_REQUIRE_UPPER "$2"
		shift 2
		;;
	--auth-password-require-number)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_PASSWORD_REQUIRE_NUMBER "$2"
		shift 2
		;;
	--auth-password-require-symbol)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_PASSWORD_REQUIRE_SYMBOL "$2"
		shift 2
		;;
	--auth-casbin-reload-interval-seconds)
		require_arg "$1" "${2:-}"
		set_app_env AUTH_CASBIN_RELOAD_INTERVAL_SECONDS "$2"
		shift 2
		;;
	--migration-auto-apply)
		require_arg "$1" "${2:-}"
		set_app_env MIGRATION_AUTO_APPLY "$2"
		shift 2
		;;
	--migration-dir)
		require_arg "$1" "${2:-}"
		set_app_env MIGRATION_DIR "$2"
		shift 2
		;;
	--cors-enabled)
		require_arg "$1" "${2:-}"
		set_app_env CORS_ENABLED "$2"
		shift 2
		;;
	--cors-allow-origins)
		require_arg "$1" "${2:-}"
		set_app_env CORS_ALLOW_ORIGINS "$2"
		shift 2
		;;
	--cors-allow-methods)
		require_arg "$1" "${2:-}"
		set_app_env CORS_ALLOW_METHODS "$2"
		shift 2
		;;
	--cors-allow-headers)
		require_arg "$1" "${2:-}"
		set_app_env CORS_ALLOW_HEADERS "$2"
		shift 2
		;;
	--cors-expose-headers)
		require_arg "$1" "${2:-}"
		set_app_env CORS_EXPOSE_HEADERS "$2"
		shift 2
		;;
	--cors-allow-credentials)
		require_arg "$1" "${2:-}"
		set_app_env CORS_ALLOW_CREDENTIALS "$2"
		shift 2
		;;
	--cors-max-age)
		require_arg "$1" "${2:-}"
		set_app_env CORS_MAX_AGE "$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		die "unknown argument: $1"
		;;
	esac
done

[ -z "$DEPLOY_DOCKER" ] || DEPLOY_DOCKER="$(normalize_yn "$DEPLOY_DOCKER")"
DEPLOY_BUILD="$(normalize_yn "$DEPLOY_BUILD")"
DEPLOY_PULL="$(normalize_yn "$DEPLOY_PULL")"
WEBUI_CHECK="$(normalize_yn "$WEBUI_CHECK")"

if [ -z "$REPO_URL" ] && [ -n "$GITHUB_PROXY_HOST" ]; then
	REPO_URL="$(repo_url_from_github_proxy "$GITHUB_PROXY_HOST" "$REPO_SLUG")"
fi

[ "$DEPLOY_CONFIRM" = "y" ] || die "--confirm is required"
[ "$DEPLOY_DOCKER" = "y" ] || die "--docker y is required; non-Docker deployment is not implemented"

case "$DEPLOY_ENV" in
staging | production) ;;
*) die "--env must be staging or production" ;;
esac

[ -n "$HOST_CONFIG_DIR" ] || HOST_CONFIG_DIR="$DEPLOY_PATH/configs"
[ -n "$HOST_DATA_DIR" ] || HOST_DATA_DIR="$DEPLOY_PATH/data"
[ -n "$HOST_LOGS_DIR" ] || HOST_LOGS_DIR="$DEPLOY_PATH/logs"
HOST_CONFIG_FILE="${HOST_CONFIG_DIR%/}/config.yaml"

[[ "$DEPLOY_PATH" = /* ]] || die "--path must be absolute"
[[ "$DEPLOY_PATH" != "/" ]] || die "--path must not be /"
[[ "$HOST_CONFIG_DIR" = /* ]] || die "--config-dir must be absolute"
[[ "$HOST_DATA_DIR" = /* ]] || die "--data-dir must be absolute"
[[ "$HOST_LOGS_DIR" = /* ]] || die "--logs-dir must be absolute"
[ "$HOST_CONFIG_DIR" != "/" ] || die "--config-dir must not be /"
[ "$HOST_DATA_DIR" != "/" ] || die "--data-dir must not be /"
[ "$HOST_LOGS_DIR" != "/" ] || die "--logs-dir must not be /"
[[ "$APP_PORT" =~ ^[0-9]+$ ]] || die "--port must be numeric"
[[ "$APP_CONTAINER_PORT" =~ ^[0-9]+$ ]] || die "--server-port must be numeric"
validate_value REPO_URL "$REPO_URL"
validate_value REPO_REF "$REPO_REF"
validate_value REPO_SLUG "$REPO_SLUG"
validate_value GITHUB_PROXY_HOST "$GITHUB_PROXY_HOST"
validate_value DEPLOY_PATH "$DEPLOY_PATH"
validate_value HOST_CONFIG_DIR "$HOST_CONFIG_DIR"
validate_value HOST_DATA_DIR "$HOST_DATA_DIR"
validate_value HOST_LOGS_DIR "$HOST_LOGS_DIR"
validate_value HOST_CONFIG_FILE "$HOST_CONFIG_FILE"
validate_value DEPLOY_IMAGE "$DEPLOY_IMAGE"
validate_value DEPLOY_CONTAINER_NAME "$DEPLOY_CONTAINER_NAME"
validate_value APP_PORT "$APP_PORT"
validate_value APP_CONTAINER_PORT "$APP_CONTAINER_PORT"
validate_value SOURCE_DIR "$SOURCE_DIR"
validate_value REGISTRY_HOST "$REGISTRY_HOST"
validate_value REGISTRY_USERNAME "$REGISTRY_USERNAME"
validate_value REGISTRY_TOKEN "$REGISTRY_TOKEN"
validate_value WEBUI_API_BASE_URL "$WEBUI_API_BASE_URL"
validate_value WEBUI_CHECK_PATH "$WEBUI_CHECK_PATH"

webui_mount_env_key="${APP_ENV_PREFIX}_WEBUI_MOUNT_PATH"
webui_enabled_env_key="${APP_ENV_PREFIX}_WEBUI_ENABLED"
webui_mount_path="${APP_ENV[$webui_mount_env_key]:-/}"
webui_mount_path="$(normalize_webui_mount_path "$webui_mount_path")"
webui_enabled="${APP_ENV[$webui_enabled_env_key]:-true}"
case "${webui_enabled,,}" in
false | n | no | 0)
	if [ "$WEBUI_CHECK_SET" = "n" ]; then
		WEBUI_CHECK="n"
	fi
	;;
esac

if [ -z "$WEBUI_CHECK_PATH" ]; then
	if [ "$webui_mount_path" = "/" ]; then
		WEBUI_CHECK_PATH="/admin"
	else
		WEBUI_CHECK_PATH="${webui_mount_path}"
	fi
else
	[[ "$WEBUI_CHECK_PATH" == /* ]] || die "--webui-check-path must start with /"
fi

if [ "$DEPLOY_PULL" = "y" ] && [ "$DEPLOY_BUILD_SET" = "n" ]; then
	DEPLOY_BUILD="n"
fi

if [ "$DEPLOY_BUILD" = "y" ] && [ "$DEPLOY_PULL" = "y" ]; then
	die "--build y and --pull y cannot be used together"
fi

require_cmd docker
if docker compose version >/dev/null 2>&1; then
	compose=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
	compose=(docker-compose)
else
	die "docker compose or docker-compose is required"
fi

cleanup_dir=""
if [ -n "$SOURCE_DIR" ]; then
	[ -f "$SOURCE_DIR/Dockerfile" ] || die "--source-dir does not look like a go-scaffold checkout"
elif [ -f "./Dockerfile" ] && [ -f "./deploy/docker-compose.production.example.yml" ]; then
	SOURCE_DIR="$(pwd)"
else
	[ -n "$REPO_URL" ] || die "--repo is required when deploy.sh is not run from a repository checkout; use --repo, or --github-proxy-host with --repo-slug"
	cleanup_dir="$(mktemp -d "${TMPDIR:-/tmp}/go-scaffold.XXXXXX")"
	clone_repo "$REPO_URL" "$REPO_REF" "$cleanup_dir"
	SOURCE_DIR="$cleanup_dir"
fi

if [ -n "$cleanup_dir" ]; then
	trap 'rm -rf "$cleanup_dir"' EXIT
fi

COMPOSE_SOURCE="${SOURCE_DIR}/deploy/docker-compose.production.example.yml"
CONFIG_SOURCE="${SOURCE_DIR}/deploy/config.production.example.yaml"
COMPOSE_FILE="docker-compose.yml"
SERVICE_NAME="go-scaffold"
HEALTH_URL="http://127.0.0.1:${APP_PORT}/health"
READY_URL="http://127.0.0.1:${APP_PORT}/ready"

[ -f "$COMPOSE_SOURCE" ] || die "compose template not found: $COMPOSE_SOURCE"
[ -f "$CONFIG_SOURCE" ] || die "config template not found: $CONFIG_SOURCE"

mkdir -p "$DEPLOY_PATH" "$HOST_CONFIG_DIR" "$HOST_DATA_DIR" "$HOST_LOGS_DIR"
if [ ! -f "$HOST_CONFIG_FILE" ]; then
	cp "$CONFIG_SOURCE" "$HOST_CONFIG_FILE"
	log "wrote default config template"
else
	log "kept existing config file"
fi
cp "$COMPOSE_SOURCE" "$DEPLOY_PATH/$COMPOSE_FILE"

if [ "$(id -u)" = "0" ]; then
	chown -R 10001:10001 "$HOST_DATA_DIR" "$HOST_LOGS_DIR"
elif command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then
	sudo chown -R 10001:10001 "$HOST_DATA_DIR" "$HOST_LOGS_DIR"
else
	log "warning: cannot chown data/logs to 10001:10001 without passwordless sudo"
fi

export DEPLOY_IMAGE DEPLOY_CONTAINER_NAME APP_PORT APP_CONTAINER_PORT HOST_CONFIG_FILE HOST_DATA_DIR HOST_LOGS_DIR
for key in "${!APP_ENV[@]}"; do
	export "$key=${APP_ENV[$key]}"
done

if [ -n "$REGISTRY_USERNAME" ] || [ -n "$REGISTRY_TOKEN" ]; then
	[ -n "$REGISTRY_USERNAME" ] || die "--registry-username is required with --registry-token"
	[ -n "$REGISTRY_TOKEN" ] || die "--registry-token is required with --registry-username"
	printf '%s' "$REGISTRY_TOKEN" | docker login "$REGISTRY_HOST" -u "$REGISTRY_USERNAME" --password-stdin >/dev/null
	log "registry login completed"
fi

log "source: $SOURCE_DIR"
if [ -n "$GITHUB_PROXY_HOST" ]; then
	log "github proxy host: $GITHUB_PROXY_HOST"
fi
log "target: $DEPLOY_PATH"
log "config file: $HOST_CONFIG_FILE"
log "data dir: $HOST_DATA_DIR"
log "logs dir: $HOST_LOGS_DIR"
log "environment: $DEPLOY_ENV"
log "image: $DEPLOY_IMAGE"
log "host port: $APP_PORT"
log "container port: $APP_CONTAINER_PORT"
if [ -n "$WEBUI_API_BASE_URL" ]; then
	log "webui api base url: $WEBUI_API_BASE_URL"
else
	log "webui api base url: same-origin"
fi
if [ "$WEBUI_CHECK" = "y" ]; then
	log "webui check path: $WEBUI_CHECK_PATH"
fi
if [ "${#APP_ENV[@]}" -gt 0 ]; then
	keys=("${!APP_ENV[@]}")
	IFS=,
	log "application env keys: ${keys[*]}"
	unset IFS
fi

if [ "$DEPLOY_BUILD" = "y" ]; then
	log "building image"
	docker build \
		--build-arg "VITE_PUBLIC_API_BASE_URL=$WEBUI_API_BASE_URL" \
		-t "$DEPLOY_IMAGE" "$SOURCE_DIR"
fi

cd "$DEPLOY_PATH"
if [ "$DEPLOY_PULL" = "y" ]; then
	log "pulling image"
	docker pull "$DEPLOY_IMAGE"
	"${compose[@]}" -f "$COMPOSE_FILE" pull "$SERVICE_NAME"
fi

log "starting service"
"${compose[@]}" -f "$COMPOSE_FILE" up -d "$SERVICE_NAME"

if command -v curl >/dev/null 2>&1; then
	check_url() {
		local name="$1"
		local url="$2"

		for _ in $(seq 1 30); do
			if curl -fsS --max-time 5 "$url" >/dev/null; then
				log "$name check passed"
				return 0
			fi
			sleep 2
		done

		die "$name check failed: $url"
	}

	check_url health "$HEALTH_URL"
	check_url ready "$READY_URL"
	if [ "$WEBUI_CHECK" = "y" ]; then
		check_url webui "http://127.0.0.1:${APP_PORT}${WEBUI_CHECK_PATH}"
	fi
else
	log "warning: curl not found on host; skipped health/ready/webui checks"
fi

log "deployment finished"
