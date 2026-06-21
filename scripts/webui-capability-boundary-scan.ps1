Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    $failures.Add($Message) | Out-Null
}

function To-RelativePath {
    param([string]$Path)
    if (Test-Path -LiteralPath $Path) {
        $fullPath = (Resolve-Path -LiteralPath $Path).Path
    } else {
        $fullPath = [System.IO.Path]::GetFullPath($Path)
    }
    $basePath = $root.TrimEnd("\", "/")
    if ($fullPath.StartsWith($basePath, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $fullPath.Substring($basePath.Length).TrimStart("\", "/").Replace("\", "/")
    }
    return $fullPath.Replace("\", "/")
}

function Read-Text {
    param([string]$Path)
    return [System.IO.File]::ReadAllText($Path, [System.Text.UTF8Encoding]::new($false, $true))
}

function Assert-FileContains {
    param(
        [string]$RelativePath,
        [string]$Needle,
        [string]$Reason
    )
    $path = Join-Path $root $RelativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        Add-Failure "$Reason missing file: $RelativePath"
        return
    }
    $text = Read-Text $path
    if (-not $text.Contains($Needle)) {
        Add-Failure "$Reason in $RelativePath"
    }
}

function Assert-NoPatternInFiles {
    param(
        [object[]]$Files,
        [string]$Pattern,
        [string]$Reason
    )
    foreach ($file in $Files) {
        $text = Read-Text $file.FullName
        foreach ($match in [regex]::Matches($text, $Pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)) {
            Add-Failure "$Reason in $(To-RelativePath $file.FullName): $($match.Value)"
        }
    }
}

$webSourceRoot = Join-Path $root "web/app/src"
if (-not (Test-Path -LiteralPath $webSourceRoot -PathType Container)) {
    Add-Failure "WebUI source root is missing: web/app/src"
}

$webSourceFiles = @()
if (Test-Path -LiteralPath $webSourceRoot -PathType Container) {
    $webSourceFiles = Get-ChildItem -LiteralPath $webSourceRoot -Recurse -File |
        Where-Object { $_.Extension -in @(".ts", ".tsx", ".json", ".css") }
}

$forbiddenCapabilityPatterns = @(
    @("(/api/v1/plugins|/plugin-api/v1|\bplugins?\b|pluginApi|PluginPage|\u63D2\u4EF6)", "forbidden plugin production surface in WebUI source"),
    @("(external\s+deploy|external\s+deployment|deployment\s+executor|deploy\s+executor|\u5916\u90E8\u90E8\u7F72|\u90E8\u7F72\u6267\u884C\u5668)", "forbidden external deployment production surface in WebUI source"),
    @("(bucket\s+lifecycle|bucket\s+policy|storage\s+policy|storagePolicy|bucketPolicy|\u751F\u547D\u5468\u671F\u7B56\u7565|\u6743\u9650\u7B56\u7565)", "forbidden bucket lifecycle or policy production surface in WebUI source"),
    @("(\bmock\b|\bfake\b|\bstub\b|\bdummy\b)", "forbidden mock or fake production capability in WebUI source")
)

foreach ($entry in $forbiddenCapabilityPatterns) {
    Assert-NoPatternInFiles -Files $webSourceFiles -Pattern $entry[0] -Reason $entry[1]
}

foreach ($needle in @(
        'serverStatus: `${systemPath}/server-status`',
        'versionPackages: `${systemPath}/version-packages`',
        'versionPackageReleases: `${systemPath}/version-packages/releases`',
        'versionPackagePublish: (idValue: EndpointID)',
        'versionPackageRollback: (idValue: EndpointID)',
        'mediaAssets: `${systemPath}/media-assets`',
        'mediaAssetUpload: `${systemPath}/media-assets/upload`',
        'operationRecordSummary: `${systemPath}/operation-records/summary`',
        'storageObjects: `${systemPath}/storage-objects`',
        'trafficProbeTargets: `${systemPath}/traffic-probes/targets`',
        'trafficProbeRun: (idValue: EndpointID)',
        'trafficProbeResults: `${systemPath}/traffic-probes/results`',
        'trafficProbeAlerts: `${systemPath}/traffic-probes/alerts`',
        'trafficProbeAlertAck: (idValue: EndpointID)',
        'trafficProbeAlertResolve: (idValue: EndpointID)'
    )) {
    Assert-FileContains "web/app/src/lib/api/endpoints.ts" $needle "WebUI System API endpoints must stay centralized and explicit"
}

foreach ($needle in @(
        'optional(apiGet<ServerStatus>(endpoints.system.serverStatus))',
        'optional(apiGet<VersionPackageEntry[]>(endpoints.system.versionPackages))',
        'optional(apiGet<VersionReleaseEventEntry[]>(endpoints.system.versionPackageReleases))',
        'optional(apiGet<MediaAssetEntry[]>(endpoints.system.mediaAssets))',
        'optional(apiGet<OperationRecordSummary>(endpoints.system.operationRecordSummary, { top_limit: 5 }))',
        'optional(apiGet<StorageObjectEntry[]>(endpoints.system.storageObjects, { limit: 50 }))',
        'optional(apiGet<TrafficProbeTarget[]>(endpoints.system.trafficProbeTargets))',
        'optional(apiGet<TrafficProbeResult[]>(endpoints.system.trafficProbeResults, { limit: 20 }))',
        'optional(apiGet<TrafficProbeAlert[]>(endpoints.system.trafficProbeAlerts, { limit: 20 }))',
        'apiPostForm<MediaAssetEntry>(endpoints.system.mediaAssetUpload',
        'apiDelete(endpoints.system.storageObjects',
        'apiPost(endpoints.system.trafficProbeRun',
        'apiPost(endpoints.system.trafficProbeAlertAck',
        'apiPost(endpoints.system.trafficProbeAlertResolve',
        'metrics.network_received_bytes',
        'metrics.network_transmitted_bytes',
        't("admin.realMetricsOnly")',
        'function resourceMetricValues(metrics: NonNullable<ServerStatus["metrics"]>, t: TFn)'
    )) {
    Assert-FileContains "web/app/src/main.tsx" $needle "WebUI System panel must consume Rust-backed API and typed server metrics"
}

foreach ($needle in @(
        '/api/v1/system/server-status',
        '/api/v1/system/version-packages',
        '/api/v1/system/version-packages/releases',
        '/api/v1/system/version-packages/{id}/publish',
        '/api/v1/system/version-packages/{id}/rollback',
        '/api/v1/system/media-assets',
        '/api/v1/system/media-assets/upload',
        '/api/v1/system/operation-records/summary',
        '/api/v1/system/storage-objects',
        '/api/v1/system/traffic-probes/targets',
        '/api/v1/system/traffic-probes/targets/{id}/run',
        '/api/v1/system/traffic-probes/results',
        '/api/v1/system/traffic-probes/alerts',
        '/api/v1/system/traffic-probes/alerts/{id}/ack',
        '/api/v1/system/traffic-probes/alerts/{id}/resolve'
    )) {
    Assert-FileContains "crates/core/app/src/transport/http/route_registry.rs" $needle "WebUI System endpoint must exist in the Rust route registry"
}

Assert-FileContains "web/app/src/i18n/locales/zh-CN.json" '"realMetricsOnly":' "zh-CN locale must state real metrics boundary"
Assert-FileContains "web/app/src/i18n/locales/en.json" '"realMetricsOnly": "Server status only shows fields collected by the backend, not metrics without collectors."' "en locale must state real metrics boundary"
Assert-FileContains "web/app/src/i18n/locales/zh-CN.json" 'Rust API catalog' "zh-CN locale must state Rust API catalog capability boundary"
Assert-FileContains "web/app/src/i18n/locales/en.json" '"description": "Only production capabilities present in the Rust API catalog are shown."' "en locale must state Rust API catalog capability boundary"

if ($failures.Count -gt 0) {
    Write-Host "WebUI capability boundary scan failed:"
    foreach ($failure in $failures) {
        Write-Host " - $failure"
    }
    exit 1
}

Write-Host "WebUI capability boundary scan passed: System UI only consumes Rust route-registry capabilities and forbidden old or mock production surfaces are absent."
