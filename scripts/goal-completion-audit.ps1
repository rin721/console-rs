param(
    [string]$Root = (Join-Path $PSScriptRoot ".."),
    [string]$TargetAcceptanceReport = "",
    [switch]$RequireReady
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$utf8Strict = [System.Text.UTF8Encoding]::new($false, $true)
$checks = New-Object System.Collections.Generic.List[object]

function Read-Text {
    param([string]$Path)
    return [System.IO.File]::ReadAllText($Path, $utf8Strict)
}

function Join-Root {
    param([string]$Path)
    return (Join-Path $rootPath $Path)
}

function Add-Check {
    param(
        [string]$Name,
        [bool]$Passed,
        [string]$Evidence,
        [string]$Next = "",
        [bool]$Required = $true
    )
    $checks.Add([pscustomobject]@{
            name = $Name
            status = if ($Passed) { "passed" } else { "failed" }
            required = $Required
            evidence = $Evidence
            next = $Next
        }) | Out-Null
}

function Invoke-AuditScript {
    param(
        [string]$RelativePath,
        [string[]]$Arguments = @()
    )
    $scriptPath = Join-Root $RelativePath
    if (-not (Test-Path -LiteralPath $scriptPath -PathType Leaf)) {
        return [pscustomobject]@{
            passed = $false
            output = "missing script: $RelativePath"
        }
    }
    $output = & powershell -NoProfile -ExecutionPolicy Bypass -File $scriptPath @Arguments 2>&1
    $exitCode = $LASTEXITCODE
    return [pscustomobject]@{
        passed = ($exitCode -eq 0)
        output = (($output | ForEach-Object { "$_" }) -join "`n").Trim()
    }
}

$goalAudit = Invoke-AuditScript "scripts/goal-deliverable-audit.ps1" @("-Root", $rootPath)
Add-Check `
    -Name "goal-deliverables" `
    -Passed $goalAudit.passed `
    -Evidence $goalAudit.output `
    -Next "Fix workspace, docs, migration matrix, skills, route registry, OpenAPI, WebUI, or reference-directory evidence."

$stageAudit = Invoke-AuditScript "scripts/stage-acceptance-report-audit.ps1" @("-Root", $rootPath)
Add-Check `
    -Name "stage-report-structure" `
    -Passed $stageAudit.passed `
    -Evidence $stageAudit.output `
    -Next "Restore required report sections, verification commands, residual risks, and below-100 boundary."

$validatorSmoke = Invoke-AuditScript "scripts/target-acceptance-report-validator-smoke.ps1"
Add-Check `
    -Name "target-report-validator-policy" `
    -Passed $validatorSmoke.passed `
    -Evidence $validatorSmoke.output `
    -Next "Fix the target report validator so incomplete or unsafe reports are rejected."

$stagePath = Join-Root "docs/ai/stage-acceptance-report.md"
if (Test-Path -LiteralPath $stagePath -PathType Leaf) {
    $stageText = Read-Text $stagePath
    $progress = $null
    $progressMatch = [regex]::Match($stageText, '\*\*(?<progress>[0-9]+(?:\.[0-9]+)?)%\*\*')
    if ($progressMatch.Success) {
        $progress = [double]::Parse($progressMatch.Groups["progress"].Value, [System.Globalization.CultureInfo]::InvariantCulture)
    }
    Add-Check `
        -Name "stage-progress-final" `
        -Passed ($null -ne $progress -and $progress -ge 100.0) `
        -Evidence $(if ($null -eq $progress) { "stage progress marker missing" } else { "progress=$progress" }) `
        -Next "Only update the stage report to 100 percent after final target evidence is complete."

    $notFinalMarkers = @(
        'PostgreSQL/MySQL external smoke',
        'target production'
    )
    $presentMarkers = @($notFinalMarkers | Where-Object { $stageText.Contains($_) })
    Add-Check `
        -Name "stage-report-no-not-final-markers" `
        -Passed ($presentMarkers.Count -eq 0) `
        -Evidence $(if ($presentMarkers.Count -eq 0) { "no not-final markers found" } else { "markers=" + ($presentMarkers -join "; ") }) `
        -Next "Remove not-final markers after target evidence and remaining capability decisions are complete."
} else {
    Add-Check `
        -Name "stage-progress-final" `
        -Passed $false `
        -Evidence "missing docs/ai/stage-acceptance-report.md" `
        -Next "Restore the stage acceptance report."
    Add-Check `
        -Name "stage-report-no-not-final-markers" `
        -Passed $false `
        -Evidence "missing docs/ai/stage-acceptance-report.md" `
        -Next "Restore the stage acceptance report."
}

$targetDocPath = Join-Root "docs/deployment/target-environment-acceptance.md"
if (Test-Path -LiteralPath $targetDocPath -PathType Leaf) {
    $targetText = Read-Text $targetDocPath
    $placeholder = [string]([char]0x5f85) + [string]([char]0x586b) + [string]([char]0x5199)
    $placeholderCount = ([regex]::Matches($targetText, [regex]::Escape($placeholder))).Count
    Add-Check `
        -Name "target-acceptance-document-filled" `
        -Passed ($placeholderCount -eq 0) `
        -Evidence "placeholder_count=$placeholderCount" `
        -Next "Run target acceptance in the real target environment, then fill the target checklist."
} else {
    Add-Check `
        -Name "target-acceptance-document-filled" `
        -Passed $false `
        -Evidence "missing docs/deployment/target-environment-acceptance.md" `
        -Next "Restore the target acceptance checklist."
}

if ([string]::IsNullOrWhiteSpace($TargetAcceptanceReport)) {
    Add-Check `
        -Name "target-acceptance-report-validated" `
        -Passed $false `
        -Evidence "TargetAcceptanceReport was not provided" `
        -Next "Provide a real target JSON report validated without AllowLocalHttp."
} else {
    $reportPath = if ([System.IO.Path]::IsPathRooted($TargetAcceptanceReport)) {
        $TargetAcceptanceReport
    } else {
        Join-Path $rootPath $TargetAcceptanceReport
    }
    if (-not (Test-Path -LiteralPath $reportPath -PathType Leaf)) {
        Add-Check `
            -Name "target-acceptance-report-validated" `
            -Passed $false `
            -Evidence "missing report: $TargetAcceptanceReport" `
            -Next "Run target-environment-acceptance.ps1 first to generate the final target report."
    } else {
        $validation = Invoke-AuditScript "scripts/validate-target-acceptance-report.ps1" @("-ReportPath", $reportPath)
        Add-Check `
            -Name "target-acceptance-report-validated" `
            -Passed $validation.passed `
            -Evidence $validation.output `
            -Next "Fix the target environment or report until it validates as full/passed non-local HTTPS evidence."
    }
}

$checkArray = @()
foreach ($check in $checks) {
    $checkArray += $check
}
$blocking = @($checkArray | Where-Object { $_.required -and $_.status -ne "passed" })
$result = [ordered]@{
    schema_version = 1
    ready = ($blocking.Count -eq 0)
    generated_at = (Get-Date).ToUniversalTime().ToString("o")
    root = $rootPath
    blocking_count = $blocking.Count
    checks = $checkArray
}

$json = $result | ConvertTo-Json -Depth 8
[Console]::Out.WriteLine($json)

if ($RequireReady -and $blocking.Count -gt 0) {
    exit 1
}
