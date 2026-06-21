#!/usr/bin/env python3
"""Build release packages.

It intentionally uses only the Python standard library so it can run in CI and
local shells without an extra virtual environment.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import zipfile
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Sequence


DEFAULT_TARGETS = ("linux/amd64", "windows/amd64", "darwin/amd64")
DEFAULT_OUTPUT = "build/releases"
DEFAULT_WEBUI_BASE_URL = "/"
DEFAULT_WEBUI_API_BASE_URL = ""
DEFAULT_BINARY_NAME = "aoi-server"
WEBUI_DIST = Path("web/app/build/client")


@dataclass(frozen=True)
class Target:
    goos: str
    goarch: str

    @classmethod
    def parse(cls, value: str) -> "Target":
        parts = [part.strip() for part in value.split("/") if part.strip()]
        if len(parts) != 2:
            raise ValueError(f"invalid target {value!r}; expected goos/goarch")
        return cls(parts[0], parts[1])

    @property
    def label(self) -> str:
        return f"{self.goos}/{self.goarch}"

    @property
    def archive_ext(self) -> str:
        return ".zip" if self.goos == "windows" else ".tar.gz"

    @property
    def binary_name(self) -> str:
        return DEFAULT_BINARY_NAME + (".exe" if self.goos == "windows" else "")


@dataclass
class Options:
    targets: list[Target]
    output: Path
    cgo: bool
    skip_web_build: bool
    webui_build_base_url: str
    webui_api_base_url: str
    clean: bool
    dry_run: bool
    verbose: bool
    version: str


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def app_metadata(root: Path, version_override: str | None) -> tuple[str, str]:
    name = os.environ.get("AOI_PACKAGE_NAME", DEFAULT_BINARY_NAME).strip() or DEFAULT_BINARY_NAME
    version = version_override or os.environ.get("AOI_VERSION", "").strip() or git_commit(root)
    return name, version


def git_commit(root: Path) -> str:
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--short=12", "HEAD"],
            cwd=root,
            check=True,
            capture_output=True,
            text=True,
        )
        return result.stdout.strip()
    except Exception:
        return "unknown"


def parse_args(argv: Sequence[str]) -> Options:
    parser = argparse.ArgumentParser(description="Build release packages")
    parser.add_argument("--target", action="append", default=[], help="Target platform in goos/goarch format")
    parser.add_argument("--output", default=DEFAULT_OUTPUT, help="Release package output directory")
    parser.add_argument("--cgo", action="store_true", help="Build with CGO_ENABLED=1")
    parser.add_argument("--skip-web-build", action="store_true", help="Skip React WebUI build and package existing static dist when present")
    parser.add_argument("--webui-build-base-url", default=DEFAULT_WEBUI_BASE_URL, help="Public base URL recorded for the React WebUI package")
    parser.add_argument("--webui-api-base-url", default=DEFAULT_WEBUI_API_BASE_URL, help="VITE_PUBLIC_API_BASE_URL for React WebUI build")
    parser.add_argument("--clean", action="store_true", help="Clean the output directory before packaging")
    parser.add_argument("--dry-run", action="store_true", help="Print planned actions without writing release packages")
    parser.add_argument("--verbose", action="store_true", help="Print command details")
    parser.add_argument("--version", default="", help="Override release version used in archive and manifest names")
    args = parser.parse_args(argv)

    target_values = expand_targets(args.target or list(DEFAULT_TARGETS))
    return Options(
        targets=[Target.parse(value) for value in target_values],
        output=Path(args.output),
        cgo=args.cgo,
        skip_web_build=args.skip_web_build,
        webui_build_base_url=args.webui_build_base_url or DEFAULT_WEBUI_BASE_URL,
        webui_api_base_url=args.webui_api_base_url or DEFAULT_WEBUI_API_BASE_URL,
        clean=args.clean,
        dry_run=args.dry_run,
        verbose=args.verbose,
        version=args.version.strip(),
    )


def expand_targets(values: Sequence[str]) -> list[str]:
    expanded: list[str] = []
    for value in values:
        expanded.extend(part.strip() for part in value.split(",") if part.strip())
    return expanded or list(DEFAULT_TARGETS)


def run_command(root: Path, command: Sequence[str], *, cwd: Path | None = None, env: dict[str, str] | None = None, dry_run: bool = False, verbose: bool = False) -> None:
    display_cwd = cwd or root
    if dry_run or verbose:
        print(f"+ ({display_cwd}) {' '.join(command)}")
    if dry_run:
        return
    next_env = os.environ.copy()
    if env:
        next_env.update(env)
    subprocess.run(command, cwd=display_cwd, env=next_env, check=True)


def webui_dist_exists(root: Path) -> bool:
    return (root / WEBUI_DIST / "index.html").is_file()


def generate_webui(root: Path, opts: Options) -> bool:
    if opts.skip_web_build:
        include = webui_dist_exists(root)
        if include:
            print("React WebUI static dist found; it will be included.")
        else:
            print("React WebUI static dist not found; packaging backend-only release.")
        return include

    print("Building React WebUI static files...")
    run_command(
        root,
        ["pnpm", "build"],
        cwd=root / "web/app",
        env={
            "VITE_PUBLIC_API_BASE_URL": opts.webui_api_base_url,
        },
        dry_run=opts.dry_run,
        verbose=opts.verbose,
    )
    if opts.dry_run:
        return True
    if not webui_dist_exists(root):
        raise RuntimeError(f"react webui static dist missing: {root / WEBUI_DIST / 'index.html'}")
    return True


def build_binary(root: Path, staging: Path, target: Target, opts: Options) -> Path:
    binary = staging / f"{target.goos}_{target.goarch}_{target.binary_name}"
    run_command(
        root,
        ["go", "build", "-mod=readonly", "-trimpath", "-ldflags=-s -w", "-o", str(binary), "./cmd/aoi"],
        env={
            "GOOS": target.goos,
            "GOARCH": target.goarch,
            "CGO_ENABLED": "1" if opts.cgo else "0",
        },
        dry_run=opts.dry_run,
        verbose=opts.verbose,
    )
    return binary


def package_name(app_name: str, version: str, target: Target) -> str:
    return f"{app_name}_{version}_{target.goos}_{target.goarch}"


def copy_runtime(root: Path, package_root: Path, binary: Path, target: Target, include_webui: bool, opts: Options, manifest: dict) -> None:
    copy_file(binary, package_root / target.binary_name)
    if target.goos != "windows":
        chmod_executable(package_root / target.binary_name)

    copy_file(root / "deploy/config.production.example.yaml", package_root / "configs/config.yaml")
    copy_file(root / "configs/config.example.yaml", package_root / "configs/config.example.yaml")
    copy_tree(root / "configs/locales", package_root / "configs/locales")
    copy_tree(root / "internal/migrations", package_root / "internal/migrations")
    if include_webui:
        copy_tree(root / WEBUI_DIST, package_root / WEBUI_DIST)
    (package_root / "data").mkdir(parents=True, exist_ok=True)
    (package_root / "logs").mkdir(parents=True, exist_ok=True)
    write_release_readme(package_root, include_webui, opts)


def copy_file(src: Path, dst: Path) -> None:
    if not src.is_file():
        raise FileNotFoundError(f"required file missing: {src}")
    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dst)


def copy_tree(src: Path, dst: Path) -> None:
    if not src.is_dir():
        raise FileNotFoundError(f"required directory missing: {src}")
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(src, dst, symlinks=False, ignore=shutil.ignore_patterns(".gitkeep"))


def chmod_executable(path: Path) -> None:
    path.chmod(path.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)


def write_release_readme(package_root: Path, include_webui: bool, opts: Options) -> None:
    lines = [
        "Release package",
        "",
        "Run:",
        f"  ./{DEFAULT_BINARY_NAME} server --config=./configs/config.yaml",
        "",
        "Windows:",
        rf"  .\{DEFAULT_BINARY_NAME}.exe server --config=.\configs\config.yaml",
        "",
        "This package includes React WebUI static files under web/app/build/client." if include_webui else "This package does not include React WebUI static files.",
        f"CGO_ENABLED={'1' if opts.cgo else '0'}.",
    ]
    if not opts.cgo:
        lines.append("With CGO disabled, SQLite is not available at runtime; use MySQL/Postgres or rebuild with --cgo.")
    lines.extend(
        [
            f"React public base URL: {opts.webui_build_base_url}",
            f"React API base URL: {opts.webui_api_base_url or '(same origin)'}",
            "",
        ]
    )
    (package_root / "README.txt").write_text("\n".join(lines), encoding="utf-8")


def write_json(path: Path, payload: dict) -> None:
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def build_file_manifest(root: Path) -> list[dict]:
    entries: list[dict] = []
    for path in sorted(root.rglob("*")):
        if not path.is_file():
            continue
        rel = path.relative_to(root).as_posix()
        if rel == "manifest.json":
            continue
        entries.append({"path": rel, "size": path.stat().st_size, "sha256": sha256_file(path)})
    return entries


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def create_archive(package_root: Path, archive_path: Path, target: Target) -> None:
    archive_path.parent.mkdir(parents=True, exist_ok=True)
    if archive_path.exists():
        archive_path.unlink()
    if target.goos == "windows":
        with zipfile.ZipFile(archive_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for path in sorted(package_root.rglob("*")):
                rel = package_root.name + "/" + path.relative_to(package_root).as_posix()
                if path.is_dir():
                    continue
                info = zipfile.ZipInfo.from_file(path, rel)
                info.external_attr = (path.stat().st_mode & 0o777) << 16
                with path.open("rb") as handle:
                    archive.writestr(info, handle.read(), compress_type=zipfile.ZIP_DEFLATED)
    else:
        with tarfile.open(archive_path, "w:gz") as archive:
            archive.add(package_root, arcname=package_root.name)


def write_checksums(output: Path, archives: Iterable[Path], *, dry_run: bool) -> None:
    checksum_path = output / "checksums.txt"
    lines = [f"{sha256_file(path)}  {path.name}" for path in sorted(archives)]
    if dry_run:
        print(f"Would write {checksum_path}")
        for line in lines:
            print(line)
        return
    checksum_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def plan_only(root: Path, output: Path, opts: Options, app_name: str, version: str, include_webui: bool) -> None:
    print("Dry run package plan")
    print(f"Repository: {root}")
    print(f"Output: {output}")
    print(f"Version: {version}")
    print(f"Targets: {', '.join(target.label for target in opts.targets)}")
    print(f"CGO_ENABLED: {'1' if opts.cgo else '0'}")
    print(f"Include WebUI: {include_webui}")
    for target in opts.targets:
        base = package_name(app_name, version, target)
        print(f"- {target.label}: {output / (base + target.archive_ext)}")


def main(argv: Sequence[str]) -> int:
    root = repo_root()
    opts = parse_args(argv)
    app_name, version = app_metadata(root, opts.version or None)
    output = opts.output if opts.output.is_absolute() else root / opts.output
    output = output.resolve()

    if opts.clean and not opts.dry_run and output.exists():
        shutil.rmtree(output)
    if not opts.dry_run:
        output.mkdir(parents=True, exist_ok=True)

    include_webui = generate_webui(root, opts)
    if opts.dry_run:
        plan_only(root, output, opts, app_name, version, include_webui)
        return 0

    if not opts.cgo:
        print("CGO_ENABLED=0: SQLite is not available in these packages; use MySQL/Postgres or rebuild with --cgo.")

    archives: list[Path] = []
    build_time = dt.datetime.now(dt.timezone.utc).isoformat()
    commit = git_commit(root)

    with tempfile.TemporaryDirectory(prefix="aoi-package-") as temp_dir:
        staging = Path(temp_dir)
        for target in opts.targets:
            base = package_name(app_name, version, target)
            package_root = staging / base
            package_root.mkdir(parents=True)
            binary = build_binary(root, staging, target, opts)
            manifest = {
                "appName": app_name,
                "version": version,
                "gitCommit": commit,
                "builtAt": build_time,
                "target": {"goos": target.goos, "goarch": target.goarch},
                "cgoEnabled": opts.cgo,
                "webuiIncluded": include_webui,
                "webuiBuildBaseUrl": opts.webui_build_base_url,
                "webuiApiBaseUrl": opts.webui_api_base_url,
            }
            copy_runtime(root, package_root, binary, target, include_webui, opts, manifest)
            manifest["files"] = build_file_manifest(package_root)
            manifest["fileCount"] = len(manifest["files"])
            write_json(package_root / "manifest.json", manifest)
            archive_path = output / f"{base}{target.archive_ext}"
            create_archive(package_root, archive_path, target)
            archives.append(archive_path)
            print(f"Built {target.label} -> {archive_path}")

    write_checksums(output, archives, dry_run=False)
    print(f"Wrote {output / 'checksums.txt'}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main(sys.argv[1:]))
    except KeyboardInterrupt:
        print("Interrupted", file=sys.stderr)
        raise SystemExit(130)
