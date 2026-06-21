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

function From-CodePoints {
    param([int[]]$CodePoints)
    return -join ($CodePoints | ForEach-Object { [char]$_ })
}

function Assert-Contains {
    param(
        [string]$Text,
        [string]$Needle,
        [string]$Description
    )
    if (-not $Text.Contains($Needle)) {
        Add-Failure "$Description is missing"
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

$stageReportPath = Join-Root "docs/ai/stage-acceptance-report.md"
$targetAcceptancePath = Join-Root "docs/deployment/target-environment-acceptance.md"

if (-not (Test-Path -LiteralPath $stageReportPath -PathType Leaf)) {
    Add-Failure "stage acceptance report is missing"
}
if (-not (Test-Path -LiteralPath $targetAcceptancePath -PathType Leaf)) {
    Add-Failure "target environment acceptance template is missing"
}

if ($failures.Count -eq 0) {
    $report = Read-StrictUtf8 $stageReportPath
    $targetAcceptance = Read-StrictUtf8 $targetAcceptancePath

    $headings = @(
        [pscustomobject]@{ Key = "completed"; Text = From-CodePoints @(0x5DF2, 0x5B8C, 0x6210, 0x5185, 0x5BB9) },
        [pscustomobject]@{ Key = "progress"; Text = From-CodePoints @(0x8FDB, 0x5EA6, 0x767E, 0x5206, 0x6BD4) },
        [pscustomobject]@{ Key = "remaining"; Text = From-CodePoints @(0x5269, 0x4F59, 0x4EFB, 0x52A1) },
        [pscustomobject]@{ Key = "removed"; Text = From-CodePoints @(0x5220, 0x9664, 0x7684, 0x65E7, 0x8BBE, 0x8BA1) },
        [pscustomobject]@{ Key = "verification"; Text = From-CodePoints @(0x9A8C, 0x8BC1, 0x7ED3, 0x679C) },
        [pscustomobject]@{ Key = "risk"; Text = From-CodePoints @(0x6B8B, 0x7559, 0x98CE, 0x9669) },
        [pscustomobject]@{ Key = "next"; Text = From-CodePoints @(0x4E0B, 0x4E00, 0x6B65) }
    )

    $previousIndex = -1
    foreach ($heading in $headings) {
        $headingText = "## $($heading.Text)"
        $index = $report.IndexOf($headingText, [System.StringComparison]::Ordinal)
        if ($index -lt 0) {
            Add-Failure "required stage report heading missing: $($heading.Key)"
            continue
        }
        if ($index -le $previousIndex) {
            Add-Failure "stage report heading order is invalid near: $($heading.Key)"
        }
        $previousIndex = $index
    }

    $progressMatches = [regex]::Matches($report, '\*\*(?<value>\d+(?:\.\d+)?)%\*\*')
    if ($progressMatches.Count -eq 0) {
        Add-Failure "stage report progress percentage is missing"
    } else {
        $currentProgress = [double]::Parse(
            $progressMatches[0].Groups["value"].Value,
            [System.Globalization.CultureInfo]::InvariantCulture
        )
        $placeholderText = From-CodePoints @(0x5F85, 0x586B, 0x5199)
        if ($targetAcceptance.Contains($placeholderText) -and $currentProgress -ge 100.0) {
            Add-Failure "stage report must stay below 100 percent while target acceptance template still has placeholders"
        }
    }

    $finalDisclaimer = From-CodePoints @(0x4E0D, 0x662F, 0x6700, 0x7EC8, 0x5B8C, 0x6210, 0x58F0, 0x660E)
    Assert-Contains $report $finalDisclaimer "non-final completion disclaimer"

    foreach ($command in @(
        "cargo fmt --all --check",
        "cargo check --workspace",
        "cargo clippy --workspace --all-targets -- -D warnings",
        "cargo test --workspace",
        "cargo build --workspace",
        "npm --prefix web/app run typecheck",
        "npm --prefix web/app run lint:i18n",
        "npm --prefix web/app run build",
        "npm --prefix web/app run test:e2e",
        "powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-deliverable-audit.ps1",
        "powershell -NoProfile -ExecutionPolicy Bypass -File scripts/aoi-admin-source-audit.ps1",
        "powershell -NoProfile -ExecutionPolicy Bypass -File scripts/stage-acceptance-report-audit.ps1",
        "powershell -NoProfile -ExecutionPolicy Bypass -File scripts/target-acceptance-report-validator-smoke.ps1",
        "git diff --check"
    )) {
        Assert-Contains $report $command "verification command '$command'"
    }

    foreach ($needle in @(
        "Plugins",
        "Go",
        "go-scaffold",
        "go-admin",
        "aoi-admin/",
        "target-environment-acceptance.ps1",
        "validate-target-acceptance-report.ps1",
        "metrics-policy",
        "Prometheus",
        "authenticated_status=200",
        "observability-token-hash",
        "CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN",
        "PostgreSQL/MySQL",
        "WebUI",
        "CSRF"
    )) {
        Assert-Contains $report $needle "stage report evidence '$needle'"
    }

    Assert-Pattern $report 'scope=full' "target acceptance scope evidence"
    Assert-Pattern $report 'result=passed' "target acceptance result evidence"
}

if ($failures.Count -gt 0) {
    [Console]::Error.WriteLine("stage acceptance report audit failed:")
    foreach ($failure in $failures) {
        [Console]::Error.WriteLine(" - $failure")
    }
    exit 1
}

[Console]::Out.WriteLine("stage acceptance report audit passed: required report sections, verification commands, non-final progress boundary, and removed-design evidence are present.")
