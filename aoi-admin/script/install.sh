#!/usr/bin/env bash
set -euo pipefail

DEFAULT_REPO_REF="main"

die() {
	printf '[install] ERROR: %s\n' "$*" >&2
	exit 1
}

usage() {
	cat <<'USAGE'
Usage:
  curl -fsSL -o install.sh https://raw.githubusercontent.com/<owner>/<repo>/main/script/install.sh
  bash install.sh --repo https://github.com/<owner>/<repo>.git --docker y --confirm [deploy options]

GitHub proxy:
  curl -fsSL -o install.sh https://raw-githubusercontent-com-gh.helloworlds.eu.org/<owner>/<repo>/main/script/install.sh
  bash install.sh --github-proxy-host github-com-gh.helloworlds.eu.org --repo-slug <owner>/<repo> --docker y --confirm [deploy options]

This bootstrap script clones the repository, then delegates to the repository
root deploy.sh with the same arguments. Use --repo, or use --github-proxy-host
with --repo-slug, to set the source explicitly:
  --repo https://github.com/<owner>/<repo>.git
  --repo-slug <owner>/<repo>
  --ref main
  --github-proxy-host github-com-gh.helloworlds.eu.org
USAGE
}

require_arg() {
	local flag="$1"
	local value="${2:-}"
	[ -n "$value" ] || die "$flag requires a value"
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

	if git clone --depth 1 --branch "$repo_ref" "$repo_url" "$target_dir" >/dev/null 2>&1; then
		return 0
	fi

	rm -rf "$target_dir"
	git clone "$repo_url" "$target_dir" >/dev/null
	git -C "$target_dir" checkout "$repo_ref" >/dev/null
}

repo_url="${REPO_URL:-${DEPLOY_REPO_URL:-}}"
repo_ref="${REPO_REF:-${DEPLOY_REPO_REF:-$DEFAULT_REPO_REF}}"
repo_slug="${REPO_SLUG:-${DEPLOY_REPO_SLUG:-}}"
github_proxy_host="${GITHUB_PROXY_HOST:-${DEPLOY_GITHUB_PROXY_HOST:-}}"
args=()

while [ "$#" -gt 0 ]; do
	case "$1" in
	--repo)
		require_arg "$1" "${2:-}"
		repo_url="$2"
		args+=("$1" "$2")
		shift 2
		;;
	--ref)
		require_arg "$1" "${2:-}"
		repo_ref="$2"
		args+=("$1" "$2")
		shift 2
		;;
	--repo-slug)
		require_arg "$1" "${2:-}"
		repo_slug="$2"
		shift 2
		;;
	--github-proxy-host)
		require_arg "$1" "${2:-}"
		github_proxy_host="$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		args+=("$1")
		shift
		;;
	esac
done

if [ -z "$repo_url" ]; then
	if [ -n "$github_proxy_host" ]; then
		repo_url="$(repo_url_from_github_proxy "$github_proxy_host" "$repo_slug")"
	fi
fi

[ -n "$repo_url" ] || die "--repo is required; use --repo, or --github-proxy-host with --repo-slug"

require_cmd git
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/go-scaffold-install.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT

printf '[install] cloning %s (%s)\n' "$repo_url" "$repo_ref"
clone_repo "$repo_url" "$repo_ref" "$work_dir"
cd "$work_dir"

exec bash ./deploy.sh "${args[@]}"
