param(
    [string]$Root = (Join-Path $PSScriptRoot ".."),
    [string]$Config = "configs/console.example.yaml",
    [string]$Secrets = "",
    [ValidateSet("", "sqlite", "postgres", "mysql")]
    [string]$Driver = "",
    [string]$Url = "",
    [string]$BaseUrl = "",
    [switch]$ApplyMigrations,
    [switch]$SkipDatabase,
    [switch]$SkipExternalSmoke,
    [switch]$SkipHttp,
    [switch]$AllowPartial,
    [string]$MetricsScrapeTokenEnvName = "CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN",
    [string]$OutputPath = "",
    [int]$TimeoutSec = 15
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$configPath = if ([System.IO.Path]::IsPathRooted($Config)) { $Config } else { Join-Path $rootPath $Config }
$secretsPath = if ([string]::IsNullOrWhiteSpace($Secrets)) {
    ""
} elseif ([System.IO.Path]::IsPathRooted($Secrets)) {
    $Secrets
} else {
    Join-Path $rootPath $Secrets
}
$reportRoot = Join-Path $rootPath "target/target-environment-acceptance"
$timestamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")
if ([string]::IsNullOrWhiteSpace($OutputPath)) {
    $OutputPath = Join-Path $reportRoot "acceptance-$timestamp.json"
} elseif (-not [System.IO.Path]::IsPathRooted($OutputPath)) {
    $OutputPath = Join-Path $rootPath $OutputPath
}
$script:metricsScrapeToken = ""

function Get-RelativePath {
    param([string]$Path)
    if ([string]::IsNullOrWhiteSpace($Path)) {
        return ""
    }
    $fullPath = [System.IO.Path]::GetFullPath($Path)
    $basePath = $rootPath.TrimEnd("\", "/")
    if ($fullPath.StartsWith($basePath, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $fullPath.Substring($basePath.Length).TrimStart("\", "/").Replace("\", "/")
    }
    return $fullPath.Replace("\", "/")
}

function Get-PowerShellExecutable {
    if ($PSVersionTable.PSEdition -eq "Core") {
        return "pwsh"
    }
    return "powershell"
}

function Protect-Text {
    param([string]$Text)
    if ($null -eq $Text) {
        return ""
    }
    $masked = $Text
    if (-not [string]::IsNullOrWhiteSpace($Url)) {
        $masked = $masked.Replace($Url, "<database-url>")
    }
    if (-not [string]::IsNullOrWhiteSpace($script:metricsScrapeToken)) {
        $masked = $masked.Replace($script:metricsScrapeToken, "<metrics-scrape-token>")
    }
    return $masked
}

function Ensure-SqliteParent {
    if ([string]::IsNullOrWhiteSpace($Url) -or -not $Url.StartsWith("sqlite://", [System.StringComparison]::OrdinalIgnoreCase)) {
        return
    }
    $dbPath = $Url.Substring("sqlite://".Length)
    if ([string]::IsNullOrWhiteSpace($dbPath) -or $dbPath -eq ":memory:") {
        return
    }
    if (-not [System.IO.Path]::IsPathRooted($dbPath)) {
        $dbPath = Join-Path $rootPath $dbPath
    }
    $parent = Split-Path -Parent $dbPath
    if (-not [string]::IsNullOrWhiteSpace($parent)) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
}

function Invoke-CheckedProcess {
    param(
        [string]$FilePath,
        [string[]]$Arguments
    )
    $output = & $FilePath @Arguments 2>&1
    $exitCode = $LASTEXITCODE
    $text = (($output | ForEach-Object { "$_" }) -join "`n").Trim()
    if ($exitCode -ne 0) {
        throw "command failed with exit code ${exitCode}: $FilePath $($Arguments -join ' ')`n$(Protect-Text $text)"
    }
    return (Protect-Text $text)
}

function Invoke-AcceptanceStep {
    param(
        [string]$Area,
        [string]$Name,
        [scriptblock]$Block
    )
    $startedAt = (Get-Date).ToUniversalTime()
    $step = [ordered]@{
        name = $Name
        status = "failed"
        started_at = $startedAt.ToString("o")
        ended_at = ""
        duration_ms = 0
        output = ""
        error = ""
    }
    try {
        $result = & $Block
        $step.status = "passed"
        if ($null -ne $result) {
            $step.output = Protect-Text (($result | ForEach-Object { "$_" }) -join "`n")
        }
    } catch {
        $script:failureCount += 1
        $step.error = Protect-Text ($_.Exception.Message)
    } finally {
        $endedAt = (Get-Date).ToUniversalTime()
        $step.ended_at = $endedAt.ToString("o")
        $step.duration_ms = [int][Math]::Round(($endedAt - $startedAt).TotalMilliseconds)
        $script:stepCount += 1
        $script:report[$Area].steps = @($script:report[$Area].steps) + [pscustomobject]$step
    }
}

function Test-LocalBaseUrlHost {
    param([string]$HostName)
    if ([string]::IsNullOrWhiteSpace($HostName)) {
        return $false
    }
    if ($HostName -in @("localhost", "127.0.0.1", "::1")) {
        return $true
    }
    $ipAddress = [System.Net.IPAddress]::None
    if ([System.Net.IPAddress]::TryParse($HostName, [ref]$ipAddress)) {
        return [System.Net.IPAddress]::IsLoopback($ipAddress)
    }
    return $false
}

function Invoke-EntrypointSecurityPolicy {
    if ($SkipHttp -or [string]::IsNullOrWhiteSpace($BaseUrl)) {
        return "http entrypoint policy skipped because HTTP probes are not requested"
    }

    try {
        $baseUri = [Uri]($BaseUrl.TrimEnd("/"))
    } catch {
        throw "BaseUrl is not a valid absolute URI: $BaseUrl"
    }
    if (-not $baseUri.IsAbsoluteUri -or [string]::IsNullOrWhiteSpace($baseUri.Host)) {
        throw "BaseUrl must be an absolute URI with host"
    }
    if ($baseUri.Scheme -notin @("http", "https")) {
        throw "BaseUrl scheme must be http or https, got $($baseUri.Scheme)"
    }

    $isLocal = Test-LocalBaseUrlHost $baseUri.Host
    if ($baseUri.Scheme -eq "http" -and -not $isLocal) {
        $message = "non-local BaseUrl uses http; target release acceptance requires https"
        $script:httpProbesEnabled = $false
        if ($AllowPartial) {
            $script:report.scope = "partial"
            return "partial target acceptance accepted for diagnostics: $message"
        }
        throw "$message. Use https:// for full target acceptance, or -AllowPartial only for diagnostics."
    }

    $tlsRequired = -not $isLocal
    $tlsOk = (-not $tlsRequired) -or $baseUri.Scheme -eq "https"
    return "entrypoint_scheme=$($baseUri.Scheme) host=$($baseUri.Host) local=$isLocal tls_required=$tlsRequired tls_ok=$tlsOk"
}

function Get-HttpResponse {
    param(
        [string]$Uri,
        [hashtable]$Headers = @{}
    )
    try {
        $response = Invoke-WebRequest -Uri $Uri -UseBasicParsing -TimeoutSec $TimeoutSec -Headers $Headers
        $contentText = if ($response.Content -is [byte[]]) {
            [System.Text.Encoding]::UTF8.GetString($response.Content)
        } else {
            [string]$response.Content
        }
        return [pscustomobject]@{
            status = [int]$response.StatusCode
            content = $contentText
        }
    } catch {
        $response = $_.Exception.Response
        if ($null -eq $response) {
            throw
        }
        if ($response.PSObject.Methods.Name -contains "GetResponseStream") {
            $stream = $response.GetResponseStream()
            $reader = New-Object System.IO.StreamReader($stream)
            try {
                $contentText = $reader.ReadToEnd()
            } finally {
                $reader.Dispose()
            }
        } elseif ($null -ne $response.Content -and $response.Content.PSObject.Methods.Name -contains "ReadAsStringAsync") {
            $contentText = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        } else {
            $contentText = ""
        }
        return [pscustomobject]@{
            status = [int]$response.StatusCode
            content = $contentText
        }
    }
}

function Invoke-HttpProbe {
    param(
        [string]$Path,
        [int]$ExpectedStatus,
        [string[]]$Contains = @()
    )
    $base = $BaseUrl.TrimEnd("/")
    $uri = "$base$Path"
    $response = Get-HttpResponse $uri
    if ($response.status -ne $ExpectedStatus) {
        throw "$Path returned $($response.status), expected $ExpectedStatus"
    }
    foreach ($needle in $Contains) {
        if (-not $response.content.Contains($needle)) {
            throw "$Path response does not contain expected text: $needle"
        }
    }
    return "status=$($response.status) path=$Path bytes=$($response.content.Length)"
}

function Invoke-JsonPostStatus {
    param(
        [string]$Path,
        [hashtable]$Headers = @{},
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession = $null
    )
    $base = $BaseUrl.TrimEnd("/")
    $uri = "$base$Path"
    try {
        $requestArgs = @{
            Uri = $uri
            Method = "Post"
            UseBasicParsing = $true
            TimeoutSec = $TimeoutSec
            ContentType = "application/json"
            Body = "{}"
            Headers = $Headers
        }
        if ($null -ne $WebSession) {
            $requestArgs.WebSession = $WebSession
        }
        $response = Invoke-WebRequest @requestArgs
        return [int]$response.StatusCode
    } catch {
        $response = $_.Exception.Response
        if ($null -eq $response) {
            throw
        }
        return [int]$response.StatusCode
    }
}

function Invoke-CsrfProbe {
    $base = $BaseUrl.TrimEnd("/")
    $publicSettingsUri = "$base/api/v1/system/public-settings"
    $publicSettings = Invoke-WebRequest -Uri $publicSettingsUri -UseBasicParsing -TimeoutSec $TimeoutSec
    if ([int]$publicSettings.StatusCode -ne 200) {
        throw "public settings returned $($publicSettings.StatusCode), expected 200"
    }

    $contentText = if ($publicSettings.Content -is [byte[]]) {
        [System.Text.Encoding]::UTF8.GetString($publicSettings.Content)
    } else {
        [string]$publicSettings.Content
    }
    $settings = $contentText | ConvertFrom-Json
    if ($settings.auth.csrf_enabled -ne $true) {
        throw "public settings reports csrf_enabled=false; target acceptance requires CSRF enabled"
    }

    $cookieName = [string]$settings.auth.csrf_cookie_name
    $headerName = [string]$settings.auth.csrf_header_name
    if ([string]::IsNullOrWhiteSpace($cookieName) -or [string]::IsNullOrWhiteSpace($headerName)) {
        throw "public settings must expose csrf_cookie_name and csrf_header_name"
    }

    $setCookieHeaders = @()
    $rawSetCookie = $publicSettings.Headers["Set-Cookie"]
    if ($null -ne $rawSetCookie) {
        if ($rawSetCookie -is [array]) {
            $setCookieHeaders += $rawSetCookie
        } else {
            $setCookieHeaders += [string]$rawSetCookie
        }
    }

    $escapedCookieName = [regex]::Escape($cookieName)
    $csrfCookie = @($setCookieHeaders | Where-Object { $_ -match "(^|,\s*)$escapedCookieName=" }) | Select-Object -First 1
    if ([string]::IsNullOrWhiteSpace($csrfCookie)) {
        throw "public settings did not set CSRF cookie '$cookieName'"
    }
    if ($csrfCookie -match "(?i)(^|;)\s*HttpOnly(\s*;|$)") {
        throw "CSRF double-submit cookie must not be HttpOnly"
    }
    if ($csrfCookie -notmatch "(?i)(^|;)\s*Secure(\s*;|$)") {
        throw "target acceptance requires Secure CSRF cookie"
    }

    $cookiePair = ([string]$csrfCookie).Split(";")[0].Trim()
    $cookieParts = $cookiePair -split "=", 2
    if ($cookieParts.Count -ne 2 -or [string]::IsNullOrWhiteSpace($cookieParts[1])) {
        throw "CSRF cookie '$cookieName' is missing a token value"
    }

    $missingStatus = Invoke-JsonPostStatus "/api/v1/auth/setup/initial-admin"
    if ($missingStatus -ne 403) {
        throw "missing CSRF POST returned $missingStatus, expected 403"
    }

    $baseUri = [Uri]$base
    $csrfSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $csrfSession.Cookies.Add(
        $baseUri,
        (New-Object System.Net.Cookie($cookieName, $cookieParts[1], "/", $baseUri.Host))
    )
    $csrfHeaders = @{}
    $csrfHeaders[$headerName] = $cookieParts[1]
    $acceptedStatus =
        Invoke-JsonPostStatus "/api/v1/auth/setup/initial-admin" $csrfHeaders $csrfSession
    if ($acceptedStatus -eq 403) {
        throw "configured CSRF cookie/header pair was still rejected with 403"
    }

    return "csrf_enabled=true cookie=$cookieName header=$headerName missing_post_status=$missingStatus configured_post_status=$acceptedStatus secure_cookie=true"
}

function Invoke-MetricsPolicyProbe {
    $path = "/api/v1/system/metrics/prometheus"
    $base = $BaseUrl.TrimEnd("/")
    $response = Get-HttpResponse "$base$path"
    if ($response.status -notin @(401, 403)) {
        throw "$path anonymous request returned $($response.status), expected 401 or 403"
    }

    foreach ($needle in @(
        "# TYPE",
        "console_cpu_usage_percent",
        "console_memory_bytes"
    )) {
        if ($response.content.Contains($needle)) {
            throw "$path anonymous response leaked Prometheus metrics body"
        }
    }

    $script:metricsScrapeToken =
        [Environment]::GetEnvironmentVariable($MetricsScrapeTokenEnvName, "Process")
    if ([string]::IsNullOrWhiteSpace($script:metricsScrapeToken)) {
        throw "metrics scrape token env '$MetricsScrapeTokenEnvName' is missing; final target acceptance must prove authorized Prometheus scrape access without putting the raw token in the report"
    }

    $headers = @{
        Authorization = "Bearer $script:metricsScrapeToken"
    }
    $authorizedResponse = Get-HttpResponse "$base$path" $headers
    if ($authorizedResponse.status -ne 200) {
        throw "$path authorized scrape request returned $($authorizedResponse.status), expected 200"
    }
    foreach ($needle in @(
        "# TYPE console_cpu_usage_percent gauge",
        "console_memory_bytes{"
    )) {
        if (-not $authorizedResponse.content.Contains($needle)) {
            throw "$path authorized scrape response does not contain expected metrics text: $needle"
        }
    }
    foreach ($forbidden in @(
        $script:metricsScrapeToken,
        "session_token_",
        "api_token_",
        "secret"
    )) {
        if (-not [string]::IsNullOrWhiteSpace($forbidden) -and $authorizedResponse.content.Contains($forbidden)) {
            throw "$path authorized scrape response leaked forbidden sensitive text"
        }
    }

    return "status=$($response.status) path=$path anonymous_rejected=true protected_metrics=true metrics_body_leaked=false authenticated_probe=passed authenticated_status=$($authorizedResponse.status) metrics_body_present=true metrics_secret_leaked=false"
}

function New-DeployPreflightArguments {
    $arguments = @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $rootPath "scripts/database-deploy-preflight.ps1"),
        "-Root",
        $rootPath,
        "-Config",
        $configPath
    )
    if (-not [string]::IsNullOrWhiteSpace($secretsPath)) {
        $arguments += @("-Secrets", $secretsPath)
    }
    if (-not [string]::IsNullOrWhiteSpace($Driver)) {
        $arguments += @("-Driver", $Driver)
    }
    if (-not [string]::IsNullOrWhiteSpace($Url)) {
        $arguments += @("-Url", $Url)
    }
    if ($ApplyMigrations) {
        $arguments += "-ApplyMigrations"
    }
    return $arguments
}

function New-ExternalSmokeArguments {
    return @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $rootPath "scripts/database-external-smoke.ps1"),
        "-Root",
        $rootPath,
        "-Driver",
        $Driver,
        "-Url",
        $Url
    )
}

$script:failureCount = 0
$script:stepCount = 0
$script:httpProbesEnabled = $true
$script:report = [ordered]@{
    schema_version = 1
    generated_at = (Get-Date).ToUniversalTime().ToString("o")
    root = $rootPath
    config = Get-RelativePath $configPath
    secrets = if ([string]::IsNullOrWhiteSpace($secretsPath)) { "" } else { Get-RelativePath $secretsPath }
    driver = if ([string]::IsNullOrWhiteSpace($Driver)) { "from-config-or-env" } else { $Driver }
    database_url = if ([string]::IsNullOrWhiteSpace($Url)) { "" } else { "<database-url>" }
    base_url = if ([string]::IsNullOrWhiteSpace($BaseUrl)) { "" } else { $BaseUrl.TrimEnd("/") }
    scope = "full"
    allow_partial = [bool]$AllowPartial
    required = [ordered]@{
        database = -not $SkipDatabase
        http = -not $SkipHttp
    }
    policy = [ordered]@{
        status = "skipped"
        steps = @()
    }
    database = [ordered]@{
        status = "skipped"
        steps = @()
    }
    http = [ordered]@{
        status = "skipped"
        steps = @()
    }
    result = "failed"
}

try {
    Push-Location $rootPath
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $OutputPath) | Out-Null

    Invoke-AcceptanceStep "policy" "acceptance-scope" {
        $partialReasons = New-Object System.Collections.Generic.List[string]
        $databaseProbeRequested = -not $SkipDatabase
        $httpProbeRequested = (-not $SkipHttp) -and (-not [string]::IsNullOrWhiteSpace($BaseUrl))
        if ($SkipDatabase) {
            $partialReasons.Add("database skipped") | Out-Null
        }
        if ($SkipHttp) {
            $partialReasons.Add("http skipped") | Out-Null
        } elseif ([string]::IsNullOrWhiteSpace($BaseUrl)) {
            $partialReasons.Add("http base url missing") | Out-Null
        }

        if (-not $databaseProbeRequested -and -not $httpProbeRequested) {
            throw "target acceptance requires at least one database or HTTP probe, even when -AllowPartial is used."
        }

        if ($partialReasons.Count -gt 0) {
            $script:report.scope = "partial"
            $reasonText = $partialReasons -join "; "
            if (-not $AllowPartial) {
                throw "full target acceptance requires database and HTTP probes; incomplete scope: $reasonText. Add -AllowPartial only for diagnostic partial reports."
            }
            return "partial target acceptance accepted for diagnostics: $reasonText"
        }

        return "full target acceptance: database and HTTP probes are required"
    }
    Invoke-AcceptanceStep "policy" "entrypoint-security" {
        Invoke-EntrypointSecurityPolicy
    }

    if (-not $SkipDatabase) {
        Ensure-SqliteParent
        $psExe = Get-PowerShellExecutable
        if ($Driver -in @("postgres", "mysql") -and -not $SkipExternalSmoke) {
            if ([string]::IsNullOrWhiteSpace($Url)) {
                $script:failureCount += 1
                $script:report.database.steps = @($script:report.database.steps) + [pscustomobject]([ordered]@{
                    name = "database-external-smoke"
                    status = "failed"
                    started_at = (Get-Date).ToUniversalTime().ToString("o")
                    ended_at = (Get-Date).ToUniversalTime().ToString("o")
                    duration_ms = 0
                    output = ""
                    error = "Driver $Driver requires -Url for database-external-smoke.ps1"
                })
            } else {
                Invoke-AcceptanceStep "database" "database-external-smoke" {
                    Invoke-CheckedProcess $psExe (New-ExternalSmokeArguments)
                }
            }
        }

        Invoke-AcceptanceStep "database" "database-deploy-preflight" {
            Invoke-CheckedProcess $psExe (New-DeployPreflightArguments)
        }
    }

    if ($script:httpProbesEnabled -and -not $SkipHttp -and -not [string]::IsNullOrWhiteSpace($BaseUrl)) {
        Invoke-AcceptanceStep "http" "health" { Invoke-HttpProbe "/health" 200 @("ok") }
        Invoke-AcceptanceStep "http" "ready" { Invoke-HttpProbe "/ready" 200 @("database") }
        Invoke-AcceptanceStep "http" "openapi" { Invoke-HttpProbe "/openapi.yaml" 200 @("openapi:") }
        Invoke-AcceptanceStep "http" "setup-status" { Invoke-HttpProbe "/api/v1/setup/status" 200 @("completed", "required_steps") }
        Invoke-AcceptanceStep "http" "public-settings" { Invoke-HttpProbe "/api/v1/system/public-settings" 200 @("product_code") }
        Invoke-AcceptanceStep "http" "metrics-policy" { Invoke-MetricsPolicyProbe }
        Invoke-AcceptanceStep "http" "csrf-policy" { Invoke-CsrfProbe }
        Invoke-AcceptanceStep "http" "webui-root" { Invoke-HttpProbe "/" 200 @("<div id=""root""></div>") }
        Invoke-AcceptanceStep "http" "webui-admin-fallback" { Invoke-HttpProbe "/admin" 200 @("<div id=""root""></div>") }
        Invoke-AcceptanceStep "http" "api-not-spa-fallback" { Invoke-HttpProbe "/api/v1/__target_acceptance_missing" 404 @() }
    }

    foreach ($areaName in @("policy", "database", "http")) {
        $areaSteps = @($script:report[$areaName].steps)
        if ($areaSteps.Count -gt 0) {
            $failedAreaSteps = @($areaSteps | Where-Object { $_.status -eq "failed" })
            $script:report[$areaName].status = if ($failedAreaSteps.Count -gt 0) { "failed" } else { "passed" }
        }
    }

    if ($script:stepCount -eq 0) {
        $script:failureCount += 1
        $script:report.result = "failed"
    } elseif ($script:failureCount -eq 0) {
        $script:report.result = if ($script:report.scope -eq "partial") { "partial" } else { "passed" }
    }
} finally {
    if ($script:report.result -notin @("passed", "partial")) {
        $script:report.result = "failed"
    }
    ($script:report | ConvertTo-Json -Depth 10) | Set-Content -LiteralPath $OutputPath -Encoding UTF8
    Pop-Location
}

if ($script:report.result -ne "passed") {
    if ($script:report.result -eq "partial") {
        Write-Host "target environment acceptance partial; diagnostic report written to $(Get-RelativePath $OutputPath)"
        exit 0
    }
    Write-Error "target environment acceptance failed; report written to $(Get-RelativePath $OutputPath)"
    exit 1
}

Write-Host "target environment acceptance passed; report written to $(Get-RelativePath $OutputPath)"
