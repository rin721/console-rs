param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$utf8Strict = [System.Text.UTF8Encoding]::new($false, $true)
$failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    $failures.Add($Message) | Out-Null
}

function Join-Root {
    param([string]$Path)
    return (Join-Path $rootPath $Path)
}

function Read-StrictUtf8 {
    param([string]$Path)
    return [System.IO.File]::ReadAllText($Path, $utf8Strict)
}

function Assert-Path {
    param(
        [string]$Path,
        [string]$Description
    )
    if (-not (Test-Path -LiteralPath (Join-Root $Path))) {
        Add-Failure "$Description is missing: $Path"
    }
}

function Assert-Contains {
    param(
        [string]$Text,
        [string]$Needle,
        [string]$Description
    )
    if (-not $Text.Contains($Needle)) {
        Add-Failure "$Description is missing from source evidence: $Needle"
    }
}

function Assert-Pattern {
    param(
        [string]$Text,
        [string]$Pattern,
        [string]$Description
    )
    if ($Text -notmatch $Pattern) {
        Add-Failure "$Description is missing"
    }
}

function From-CodePoints {
    param([int[]]$CodePoints)
    return -join ($CodePoints | ForEach-Object { [char]$_ })
}

$sourceIndexPath = Join-Root "docs/migration/aoi-admin-source-index.md"
$migrationMatrixPath = Join-Root "docs/migration/aoi-admin-migration-matrix.md"

Assert-Path "aoi-admin" "reference directory"
Assert-Path "docs/migration/aoi-admin-source-index.md" "source index"
Assert-Path "docs/migration/aoi-admin-migration-matrix.md" "migration matrix"

$referencePaths = @(
    "aoi-admin/AGENTS.md",
    "aoi-admin/.agents/skills",
    "aoi-admin/docs/ai",
    "aoi-admin/docs/ai/README.md",
    "aoi-admin/go.mod",
    "aoi-admin/internal/app",
    "aoi-admin/internal/app/initcenter",
    "aoi-admin/internal/transport/http/contracts.go",
    "aoi-admin/internal/modules/iam",
    "aoi-admin/internal/modules/system",
    "aoi-admin/internal/migrations",
    "aoi-admin/configs/config.example.yaml",
    "aoi-admin/configs/examples",
    "aoi-admin/configs/locales",
    "aoi-admin/web/app/package.json",
    "aoi-admin/web/app/app/i18n",
    "aoi-admin/web/app/app/lib/api",
    "aoi-admin/web/app/app/lib/api/plugins.ts",
    "aoi-admin/web/app/app/routes",
    "aoi-admin/web/app/tests/e2e/smoke.spec.ts",
    "aoi-admin/internal/plugin",
    "aoi-admin/pkg/plugin",
    "aoi-admin/pkg/pluginapi",
    "aoi-admin/docs/api/plugin-protocol",
    "aoi-admin/docs/modules/plugins.md",
    "aoi-admin/docs/architecture/distributed-plugin-system.md",
    "aoi-admin/_examples/remote-plugins"
)

foreach ($path in $referencePaths) {
    Assert-Path $path "required aoi-admin audit source"
}

if ($failures.Count -eq 0) {
    $sourceIndex = Read-StrictUtf8 $sourceIndexPath
    $migrationMatrix = Read-StrictUtf8 $migrationMatrixPath

    $sourceNeedles = @(
        "aoi-admin/AGENTS.md",
        "aoi-admin/.agents/skills",
        "aoi-admin/docs/ai/README.md",
        "aoi-admin/go.mod",
        "aoi-admin/internal/app",
        "aoi-admin/internal/transport/http/contracts.go",
        "aoi-admin/internal/app/initcenter",
        "aoi-admin/internal/modules/iam",
        "aoi-admin/internal/modules/system",
        "aoi-admin/internal/migrations",
        "20260615000100_create_plugin_registry.sql",
        "20260615000200_add_plugin_instance_transport.sql",
        "aoi-admin/configs/config.example.yaml",
        "configs/locales",
        "aoi-admin/web/app/package.json",
        "aoi-admin/web/app/app/i18n",
        "aoi-admin/web/app/app/lib/api",
        "plugins.ts",
        "aoi-admin/web/app/app/routes",
        "aoi-admin/web/app/tests/e2e/smoke.spec.ts",
        "aoi-admin/internal/plugin",
        "pkg/plugin",
        "pkg/pluginapi",
        "docs/api/plugin-protocol",
        "docs/modules/plugins.md",
        "distributed-plugin-system.md",
        "_examples/remote-plugins"
    )

    foreach ($needle in $sourceNeedles) {
        Assert-Contains $sourceIndex $needle "source index coverage"
    }

    $scopeNeedles = @(
        (From-CodePoints @(0x4EE3, 0x7801)),
        (From-CodePoints @(0x6587, 0x6863)),
        (From-CodePoints @(0x8DEF, 0x7531, 0x5951, 0x7EA6)),
        (From-CodePoints @(0x8FC1, 0x79FB)),
        (From-CodePoints @(0x914D, 0x7F6E)),
        "i18n",
        "React",
        "AGENTS.md",
        ".agents/skills",
        "docs/ai"
    )
    foreach ($needle in $scopeNeedles) {
        Assert-Contains $migrationMatrix $needle "migration matrix audit scope"
    }

    $deleteWord = From-CodePoints @(0x5220, 0x9664)
    $deferWord = From-CodePoints @(0x6682, 0x7F13)
    $externalMetrics = From-CodePoints @(0x5916, 0x90E8, 0x6307, 0x6807)
    $deletePluginApiClient = $deleteWord + (From-CodePoints @(0x63D2, 0x4EF6)) + " API client"

    Assert-Pattern $sourceIndex "(?s)aoi-admin/internal/plugin.+$deleteWord" "plugin code deletion evidence"
    Assert-Pattern $sourceIndex "(?s)plugins-remote-rpc\.example\.yaml.+$deleteWord" "plugin config and migration deletion evidence"
    Assert-Pattern $sourceIndex "(?s)docs/api/plugin-protocol.+$deleteWord" "plugin docs and examples deletion evidence"
    Assert-Contains $sourceIndex $deletePluginApiClient "React plugin API deletion evidence"
    Assert-Pattern $migrationMatrix "(?s)## $deleteWord.+plugin" "migration matrix plugin deletion section"
    Assert-Pattern $migrationMatrix "(?s)## $deferWord.+$externalMetrics" "migration matrix deferred external capability section"
}

if ($failures.Count -gt 0) {
    [Console]::Error.WriteLine("aoi-admin source audit failed:")
    foreach ($failure in $failures) {
        [Console]::Error.WriteLine(" - $failure")
    }
    exit 1
}

[Console]::Out.WriteLine("aoi-admin source audit passed: reference paths, source index coverage, migration matrix scope, and plugin deletion evidence are present.")
