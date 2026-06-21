param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$reportRoot = Join-Path $rootPath "target/target-acceptance-report-validator-smoke"
$validatorPath = Join-Path $rootPath "scripts/validate-target-acceptance-report.ps1"

function Get-PowerShellExecutable {
    if ($PSVersionTable.PSEdition -eq "Core") {
        return "pwsh"
    }
    return "powershell"
}

function New-Step {
    param(
        [string]$Name,
        [string]$Status = "passed",
        [string]$Output = "",
        [string]$ErrorText = ""
    )
    return [pscustomobject]([ordered]@{
        name = $Name
        status = $Status
        started_at = "2026-06-21T00:00:00Z"
        ended_at = "2026-06-21T00:00:01Z"
        duration_ms = 1000
        output = $Output
        error = $ErrorText
    })
}

function New-Report {
    param(
        [string]$BaseUrl = "https://console.example.com",
        [string]$Result = "passed",
        [string]$Scope = "full",
        [bool]$AllowPartial = $false,
        [string]$PolicyStatus = "passed",
        [string]$DatabaseStatus = "passed",
        [string]$HttpStatus = "passed",
        [string]$EntrypointOutput = "entrypoint_scheme=https host=console.example.com local=False tls_required=True tls_ok=True",
        [bool]$IncludeCsrf = $true,
        [bool]$IncludeMetricsPolicy = $true,
        [string]$MetricsPolicyOutput = "status=401 path=/api/v1/system/metrics/prometheus anonymous_rejected=true protected_metrics=true metrics_body_leaked=false authenticated_probe=passed authenticated_status=200 metrics_body_present=true metrics_secret_leaked=false"
    )

    $httpSteps = @(
        (New-Step "health" "passed" "status=200 path=/health bytes=2"),
        (New-Step "ready" "passed" "status=200 path=/ready bytes=64"),
        (New-Step "openapi" "passed" "status=200 path=/openapi.yaml bytes=100"),
        (New-Step "setup-status" "passed" "status=200 path=/api/v1/setup/status bytes=100"),
        (New-Step "public-settings" "passed" "status=200 path=/api/v1/system/public-settings bytes=100")
    )
    if ($IncludeMetricsPolicy) {
        $httpSteps += New-Step "metrics-policy" "passed" $MetricsPolicyOutput
    }
    $httpSteps += @(
        (New-Step "webui-root" "passed" "status=200 path=/ bytes=100"),
        (New-Step "webui-admin-fallback" "passed" "status=200 path=/admin bytes=100"),
        (New-Step "api-not-spa-fallback" "passed" "status=404 path=/api/v1/__target_acceptance_missing bytes=0")
    )
    if ($IncludeCsrf) {
        $httpSteps += New-Step "csrf-policy" "passed" "csrf_enabled=true cookie=console_csrf header=X-CSRF-Token missing_post_status=403 configured_post_status=422 secure_cookie=true"
    }

    return [pscustomobject]([ordered]@{
        schema_version = 1
        generated_at = "2026-06-21T00:00:00Z"
        root = $rootPath
        config = "configs/console.example.yaml"
        secrets = ""
        driver = "postgres"
        database_url = "<database-url>"
        base_url = $BaseUrl
        scope = $Scope
        allow_partial = $AllowPartial
        required = [pscustomobject]([ordered]@{
            database = $true
            http = $true
        })
        policy = [pscustomobject]([ordered]@{
            status = $PolicyStatus
            steps = @(
                (New-Step "acceptance-scope" "passed" "full target acceptance: database and HTTP probes are required"),
                (New-Step "entrypoint-security" $PolicyStatus $EntrypointOutput)
            )
        })
        database = [pscustomobject]([ordered]@{
            status = $DatabaseStatus
            steps = @(
                (New-Step "database-deploy-preflight" $DatabaseStatus "database deploy preflight passed for postgres: insert_id_read=ReturningIdInStatement, schema_ready=True, repository_ready=True, serve_ready=True.")
            )
        })
        http = [pscustomobject]([ordered]@{
            status = $HttpStatus
            steps = $httpSteps
        })
        result = $Result
    })
}

function Write-Report {
    param(
        [string]$Name,
        [object]$Report
    )
    $path = Join-Path $reportRoot $Name
    $Report | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $path -Encoding UTF8
    return $path
}

function Invoke-Validator {
    param([string[]]$Arguments)
    $psExe = Get-PowerShellExecutable
    $previousPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $output = & $psExe -NoProfile -ExecutionPolicy Bypass -File $validatorPath @Arguments 2>&1
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousPreference
    return [pscustomobject]([ordered]@{
        exit_code = $exitCode
        output = (($output | ForEach-Object { "$_" }) -join "`n")
    })
}

function Assert-ValidatorPass {
    param(
        [string]$Name,
        [string[]]$Arguments
    )
    $result = Invoke-Validator $Arguments
    if ($result.exit_code -ne 0) {
        throw "validator expected pass for ${Name}, exit=$($result.exit_code): $($result.output)"
    }
    [Console]::Out.WriteLine("validator accepted ${Name} as expected")
}

function Assert-ValidatorFail {
    param(
        [string]$Name,
        [string[]]$Arguments
    )
    $result = Invoke-Validator $Arguments
    if ($result.exit_code -eq 0) {
        throw "validator expected failure for ${Name}, but it passed"
    }
    [Console]::Out.WriteLine("validator rejected ${Name} as expected")
}

if (-not (Test-Path -LiteralPath $validatorPath -PathType Leaf)) {
    throw "validator script is missing: $validatorPath"
}

if (Test-Path -LiteralPath $reportRoot) {
    Remove-Item -LiteralPath $reportRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $reportRoot | Out-Null

$finalHttpsReport = Write-Report "final-https-pass.json" (New-Report)
$localHttpReport = Write-Report "local-http-pass.json" (New-Report `
    -BaseUrl "http://127.0.0.1:18080" `
    -EntrypointOutput "entrypoint_scheme=http host=127.0.0.1 local=True tls_required=False tls_ok=True")
$partialReport = Write-Report "partial-rejected.json" (New-Report `
    -Result "partial" `
    -Scope "partial" `
    -AllowPartial $true `
    -HttpStatus "skipped")
$failedReport = Write-Report "failed-rejected.json" (New-Report `
    -Result "failed" `
    -PolicyStatus "failed" `
    -EntrypointOutput "")
$missingCsrfReport = Write-Report "missing-csrf-rejected.json" (New-Report -IncludeCsrf $false)
$missingMetricsReport = Write-Report "missing-metrics-policy-rejected.json" (New-Report -IncludeMetricsPolicy $false)
$missingAuthenticatedMetricsReport = Write-Report "missing-authenticated-metrics-rejected.json" (New-Report `
    -MetricsPolicyOutput "status=401 path=/api/v1/system/metrics/prometheus anonymous_rejected=true protected_metrics=true metrics_body_leaked=false")

Assert-ValidatorPass "final https report" @("-ReportPath", $finalHttpsReport)
Assert-ValidatorPass "local http smoke report" @("-ReportPath", $localHttpReport, "-AllowLocalHttp")
Assert-ValidatorFail "local http final report" @("-ReportPath", $localHttpReport)
Assert-ValidatorFail "partial report" @("-ReportPath", $partialReport)
Assert-ValidatorFail "failed report" @("-ReportPath", $failedReport)
Assert-ValidatorFail "missing csrf report" @("-ReportPath", $missingCsrfReport)
Assert-ValidatorFail "missing metrics policy report" @("-ReportPath", $missingMetricsReport)
Assert-ValidatorFail "missing authenticated metrics report" @("-ReportPath", $missingAuthenticatedMetricsReport)

[Console]::Out.WriteLine("target acceptance report validator smoke passed")
