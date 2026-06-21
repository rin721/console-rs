param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$failures = New-Object System.Collections.Generic.List[string]
$utf8Strict = [System.Text.UTF8Encoding]::new($false, $true)

function Add-Failure {
    param([string]$Message)
    $failures.Add($Message) | Out-Null
}

function Get-RelativePath {
    param([string]$Path)
    $fullPath = [System.IO.Path]::GetFullPath($Path)
    $basePath = $rootPath.TrimEnd("\", "/")
    if ($fullPath.StartsWith($basePath, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $fullPath.Substring($basePath.Length).TrimStart("\", "/").Replace("\", "/")
    }
    return $fullPath.Replace("\", "/")
}

function Join-Root {
    param([string]$Path)
    return (Join-Path $rootPath $Path)
}

function Read-Text {
    param([string]$Path)
    return [System.IO.File]::ReadAllText($Path, $utf8Strict)
}

function Assert-File {
    param([string]$Path)
    $fullPath = Join-Root $Path
    if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
        Add-Failure "required file is missing: $Path"
    }
}

function Assert-Directory {
    param([string]$Path)
    $fullPath = Join-Root $Path
    if (-not (Test-Path -LiteralPath $fullPath -PathType Container)) {
        Add-Failure "required directory is missing: $Path"
    }
}

function Assert-Contains {
    param(
        [string]$Path,
        [string]$Pattern,
        [string]$Description
    )
    $fullPath = Join-Root $Path
    if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
        Add-Failure "cannot inspect missing file for ${Description}: $Path"
        return
    }
    $text = Read-Text $fullPath
    if ($text -notmatch $Pattern) {
        Add-Failure "$Description not found in $Path"
    }
}

function Assert-NoFileOutsideReference {
    param(
        [string]$Filter,
        [string]$Description
    )
    $files = Get-ChildItem -LiteralPath $rootPath -Filter $Filter -Recurse -File |
        Where-Object {
            $relative = Get-RelativePath $_.FullName
            $relative -notmatch "^(aoi-admin|target|\.git|web/app/node_modules)(/|$)"
        }
    foreach ($file in $files) {
        Add-Failure "$Description outside reference directory: $(Get-RelativePath $file.FullName)"
    }
}

$requiredFiles = @(
    "Cargo.toml",
    "Cargo.lock",
    ".env.example",
    ".gitignore",
    "README.md",
    "AGENTS.md",
    "configs/console.example.yaml",
    "configs/console.production.example.yaml",
    "configs/console.secrets.example.yaml",
    "docs/architecture/overview.md",
    "docs/api/README.md",
    "docs/api/openapi.yaml",
    "docs/ai/collaboration.md",
    "docs/ai/requirement-evidence-audit.md",
    "docs/ai/stage-acceptance-report.md",
    "docs/deployment/database-runtime-matrix.md",
    "docs/deployment/local-runbook.md",
    "docs/deployment/target-environment-acceptance.md",
    "docs/migration/aoi-admin-migration-matrix.md",
    "docs/migration/aoi-admin-source-index.md",
    "scripts/aoi-admin-source-audit.ps1",
    "scripts/stage-acceptance-report-audit.ps1",
    "scripts/goal-completion-audit.ps1",
    "migrations/README.md",
    "migrations/20260621000100_init_core.sql",
    "migrations/postgres/20260621000100_init_core.sql",
    "migrations/mysql/20260621000100_init_core.sql",
    "web/app/package.json",
    "web/app/src/i18n/index.ts",
    "web/app/src/i18n/locales/zh-CN.json",
    "web/app/src/i18n/locales/en.json",
    ".agents/skills/rust-platform/SKILL.md",
    ".agents/skills/api-contract/SKILL.md",
    ".agents/skills/frontend-quality/SKILL.md",
    ".agents/skills/lifecycle-types/SKILL.md",
    ".agents/skills/quality/SKILL.md"
)

foreach ($path in $requiredFiles) {
    Assert-File $path
}

$requiredDirectories = @(
    "aoi-admin",
    "crates/core/app/src/domain",
    "crates/core/app/src/handler",
    "crates/core/app/src/infrastructure",
    "crates/core/app/src/repository",
    "crates/core/app/src/scheduler",
    "crates/core/app/src/service",
    "crates/core/app/src/transport/http",
    "crates/core/config/src",
    "crates/core/types/src",
    "crates/tools/crypto/src",
    "web/app/src/lib/api",
    "web/app/tests/e2e"
)

foreach ($path in $requiredDirectories) {
    Assert-Directory $path
}

$workspaceMembers = @(
    "crates/core/app",
    "crates/core/config",
    "crates/core/types",
    "crates/tools/crypto"
)

foreach ($member in $workspaceMembers) {
    Assert-Contains "Cargo.toml" ([regex]::Escape($member)) "workspace member $member"
}

Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*axum\s*=' "axum dependency"
Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*tokio\s*=' "tokio dependency"
Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*sqlx\s*=' "sqlx dependency"
Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*tower-http\s*=' "tower-http dependency"
Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*tracing\s*=' "tracing dependency"
Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*clap\s*=' "clap dependency"
Assert-Contains "crates/core/config/Cargo.toml" '(?m)^\s*config-rs\s*=' "config crate dependency"
Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*thiserror\s*=' "thiserror dependency"
Assert-Contains "crates/core/app/Cargo.toml" '(?m)^\s*anyhow\s*=' "anyhow dependency"

Assert-Contains "crates/core/app/src/transport/http/route_registry.rs" 'RouteContract' "typed route registry"
Assert-Contains "crates/core/app/src/transport/http/route_registry.rs" 'openapi' "OpenAPI generation from route registry"
Assert-Contains "crates/core/app/src/transport/http/router.rs" 'route_registry::contracts' "router registration from route registry"
Assert-Contains "docs/api/openapi.yaml" 'title:\s*Aoi' "Aoi product OpenAPI title or description"
Assert-Contains "docs/api/openapi.yaml" '/api/v1/setup/status' "Setup route in OpenAPI"
Assert-Contains "docs/api/openapi.yaml" '/api/v1/auth/login' "IAM route in OpenAPI"
Assert-Contains "docs/api/openapi.yaml" '/api/v1/system/apis' "System route in OpenAPI"

$sectionMigrated = "##\s+" + [char]0x8fc1 + [char]0x79fb
$sectionRewritten = "##\s+" + [char]0x91cd + [char]0x5199
$sectionDeleted = "##\s+" + [char]0x5220 + [char]0x9664
$sectionDeferred = "##\s+" + [char]0x6682 + [char]0x7f13
Assert-Contains "docs/migration/aoi-admin-migration-matrix.md" $sectionMigrated "migration matrix migrated section"
Assert-Contains "docs/migration/aoi-admin-migration-matrix.md" $sectionRewritten "migration matrix rewritten section"
Assert-Contains "docs/migration/aoi-admin-migration-matrix.md" $sectionDeleted "migration matrix deleted section"
Assert-Contains "docs/migration/aoi-admin-migration-matrix.md" $sectionDeferred "migration matrix deferred section"
Assert-Contains "docs/migration/aoi-admin-source-index.md" 'AGENTS\.md' "old AGENTS audit source"
Assert-Contains "docs/migration/aoi-admin-source-index.md" 'docs/ai' "old docs/ai audit source"
Assert-Contains "README.md" 'Rust' "Rust product positioning"
Assert-Contains "README.md" 'aoi-admin/' "reference-only positioning in README"
Assert-Contains "AGENTS.md" 'aoi-admin/' "reference-only aoi-admin rule"

Assert-NoFileOutsideReference "*.go" "Go runtime source"

$migrationFiles = Get-ChildItem -LiteralPath (Join-Root "migrations") -Filter "*.sql" -Recurse -File
foreach ($file in $migrationFiles) {
    $text = Read-Text $file.FullName
    if ($text -match '(?i)plugin') {
        Add-Failure "plugin table or plugin schema text found in migration: $(Get-RelativePath $file.FullName)"
    }
}

$webPackage = Join-Root "web/app/package.json"
if (Test-Path -LiteralPath $webPackage -PathType Leaf) {
    $package = Get-Content -LiteralPath $webPackage -Raw -Encoding UTF8 | ConvertFrom-Json
    foreach ($scriptName in @("typecheck", "lint:i18n", "build", "test:e2e")) {
        if ($null -eq $package.scripts.$scriptName) {
            Add-Failure "web/app/package.json missing script: $scriptName"
        }
    }
}

if ($failures.Count -gt 0) {
    [Console]::Error.WriteLine("goal deliverable audit failed:")
    foreach ($failure in $failures) {
        [Console]::Error.WriteLine(" - $failure")
    }
    exit 1
}

[Console]::Out.WriteLine("goal deliverable audit passed: workspace, deliverables, migration evidence, WebUI/i18n, route registry, migrations, project skills, and forbidden Go/plugin runtime boundaries are present.")
