param(
    [Parameter(Mandatory = $true)]
    [string]$ReportPath,
    [switch]$AllowLocalHttp
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$resolvedReportPath = (Resolve-Path -LiteralPath $ReportPath).Path
$failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    $failures.Add($Message) | Out-Null
}

function Test-LocalHost {
    param([string]$HostName)
    if ([string]::IsNullOrWhiteSpace($HostName)) {
        return $false
    }
    $normalizedHost = $HostName.Trim()
    if ($normalizedHost.StartsWith("[") -and $normalizedHost.EndsWith("]")) {
        $normalizedHost = $normalizedHost.Substring(1, $normalizedHost.Length - 2)
    }
    $normalizedHost = $normalizedHost.ToLowerInvariant()
    if (@("localhost", "127.0.0.1", "::1") -contains $normalizedHost) {
        return $true
    }
    $ipAddress = [System.Net.IPAddress]::None
    if ([System.Net.IPAddress]::TryParse($normalizedHost, [ref]$ipAddress)) {
        return [System.Net.IPAddress]::IsLoopback($ipAddress)
    }
    return $false
}

function Get-AreaSteps {
    param(
        [object]$Report,
        [string]$AreaName
    )
    if ($null -eq $Report.$AreaName -or $null -eq $Report.$AreaName.steps) {
        return @()
    }
    return @($Report.$AreaName.steps)
}

function Get-RequiredStep {
    param(
        [object]$Report,
        [string]$AreaName,
        [string]$StepName
    )
    $steps = Get-AreaSteps $Report $AreaName
    $step = @($steps | Where-Object { $_.name -eq $StepName }) | Select-Object -First 1
    if ($null -eq $step) {
        Add-Failure "${AreaName}.${StepName} step is missing"
        return $null
    }
    if ($step.status -ne "passed") {
        Add-Failure "${AreaName}.${StepName} step status is '$($step.status)', expected 'passed'"
    }
    return $step
}

function Test-OutputContains {
    param(
        [object]$Step,
        [string]$Needle,
        [string]$Description
    )
    if ($null -eq $Step) {
        return
    }
    $output = [string]$Step.output
    if (-not $output.Contains($Needle)) {
        Add-Failure "$Description is missing '$Needle'"
    }
}

$report = Get-Content -LiteralPath $resolvedReportPath -Raw -Encoding UTF8 | ConvertFrom-Json
$allowLocalHttpForSmoke = $AllowLocalHttp.IsPresent

if ($report.schema_version -ne 1) {
    Add-Failure "schema_version must be 1"
}
if ($report.result -ne "passed") {
    Add-Failure "result must be 'passed', got '$($report.result)'"
}
if ($report.scope -ne "full") {
    Add-Failure "scope must be 'full', got '$($report.scope)'"
}
if ($report.allow_partial -ne $false) {
    Add-Failure "allow_partial must be false"
}
if ($report.required.database -ne $true -or $report.required.http -ne $true) {
    Add-Failure "required.database and required.http must both be true"
}

foreach ($areaName in @("policy", "database", "http")) {
    if ($null -eq $report.$areaName) {
        Add-Failure "$areaName area is missing"
        continue
    }
    if ($report.$areaName.status -ne "passed") {
        Add-Failure "$areaName status must be 'passed', got '$($report.$areaName.status)'"
    }
}

if ([string]::IsNullOrWhiteSpace([string]$report.base_url)) {
    Add-Failure "base_url must be present"
} else {
    $baseUrlText = [string]$report.base_url
    $baseUrlPattern = '^(?<scheme>https?)://(?<host>\[[^\]]+\]|[^/:]+)(:\d+)?(/.*)?$'
    if ($baseUrlText -notmatch $baseUrlPattern) {
        Add-Failure "base_url must be an absolute http or https URI: $baseUrlText"
    } else {
        $baseUrlScheme = $Matches["scheme"].ToLowerInvariant()
        $baseUrlHost = $Matches["host"]
        $isLocal = Test-LocalHost $baseUrlHost
        if ($baseUrlScheme -ne "https") {
            if ($isLocal) {
                if (-not $allowLocalHttpForSmoke) {
                    Add-Failure "base_url must use https for final target evidence; got '$baseUrlScheme'"
                }
            } else {
                Add-Failure "base_url must use https for final target evidence; got '$baseUrlScheme'"
            }
        }
        if ((-not $isLocal) -and ($baseUrlScheme -ne "https")) {
            Add-Failure "non-local base_url must use https"
        }
    }
}

$scopeStep = Get-RequiredStep $report "policy" "acceptance-scope"
Test-OutputContains $scopeStep "full target acceptance" "acceptance-scope output"

$entrypointStep = Get-RequiredStep $report "policy" "entrypoint-security"
Test-OutputContains $entrypointStep "tls_ok=True" "entrypoint-security output"
if ($null -ne $entrypointStep) {
    $entrypointOutput = [string]$entrypointStep.output
    if ($entrypointOutput.Contains("partial target acceptance accepted")) {
        Add-Failure "entrypoint-security output must not describe partial diagnostics"
    }
    if (-not $allowLocalHttpForSmoke -and $entrypointOutput.Contains("tls_required=False")) {
        Add-Failure "entrypoint-security must prove TLS is required for final non-local target evidence"
    }
}

$deployStep = Get-RequiredStep $report "database" "database-deploy-preflight"
Test-OutputContains $deployStep "serve_ready=True" "database-deploy-preflight output"
Test-OutputContains $deployStep "schema_ready=True" "database-deploy-preflight output"
Test-OutputContains $deployStep "repository_ready=True" "database-deploy-preflight output"

foreach ($stepName in @(
    "health",
    "ready",
    "openapi",
    "setup-status",
    "public-settings",
    "webui-root",
    "webui-admin-fallback",
    "api-not-spa-fallback"
)) {
    $step = Get-RequiredStep $report "http" $stepName
    if ($stepName -eq "api-not-spa-fallback") {
        Test-OutputContains $step "status=404" "api-not-spa-fallback output"
    } else {
        Test-OutputContains $step "status=200" "$stepName output"
    }
}

$csrfStep = Get-RequiredStep $report "http" "csrf-policy"
foreach ($needle in @(
    "csrf_enabled=true",
    "missing_post_status=403",
    "secure_cookie=true"
)) {
    Test-OutputContains $csrfStep $needle "csrf-policy output"
}
if ($null -ne $csrfStep -and -not ([string]$csrfStep.output -match "configured_post_status=(?!403)\d+")) {
    Add-Failure "csrf-policy must prove the configured CSRF cookie/header pair is not rejected with 403"
}

$metricsStep = Get-RequiredStep $report "http" "metrics-policy"
foreach ($needle in @(
    "anonymous_rejected=true",
    "protected_metrics=true",
    "metrics_body_leaked=false",
    "authenticated_probe=passed",
    "authenticated_status=200",
    "metrics_body_present=true",
    "metrics_secret_leaked=false"
)) {
    Test-OutputContains $metricsStep $needle "metrics-policy output"
}
if ($null -ne $metricsStep -and -not ([string]$metricsStep.output -match "status=(401|403)")) {
    Add-Failure "metrics-policy must prove anonymous Prometheus metrics access is rejected with 401 or 403"
}

if ($failures.Count -gt 0) {
    [Console]::Error.WriteLine("target acceptance report validation failed: $resolvedReportPath")
    foreach ($failure in $failures) {
        [Console]::Error.WriteLine(" - $failure")
    }
    exit 1
}

[Console]::Out.WriteLine("target acceptance report validation passed: $resolvedReportPath")
