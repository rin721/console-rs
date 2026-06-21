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

function Assert-Contains {
    param(
        [string]$Text,
        [string]$Needle,
        [string]$Message
    )
    if ($Text -notlike "*$Needle*") {
        Add-Failure $Message
    }
}

function Assert-NotContains {
    param(
        [string]$Text,
        [string]$Needle,
        [string]$Message
    )
    if ($Text -like "*$Needle*") {
        Add-Failure $Message
    }
}

$clientPath = Join-Path $root "web/app/src/lib/api/client.ts"
$typesPath = Join-Path $root "web/app/src/lib/api/types.ts"
$endpointsPath = Join-Path $root "web/app/src/lib/api/endpoints.ts"
$systemDomainPath = Join-Path $root "crates/core/app/src/domain/system.rs"
$systemServicePath = Join-Path $root "crates/core/app/src/service/system.rs"
$routeRegistryPath = Join-Path $root "crates/core/app/src/transport/http/route_registry.rs"

foreach ($path in @($clientPath, $typesPath, $endpointsPath, $systemDomainPath, $systemServicePath, $routeRegistryPath)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        Add-Failure "required runtime config boundary file is missing: $(To-RelativePath $path)"
    }
}

$clientText = Read-Text $clientPath
$typesText = Read-Text $typesPath
$systemDomainText = Read-Text $systemDomainPath
$systemServiceText = Read-Text $systemServicePath
$routeRegistryText = Read-Text $routeRegistryPath

foreach ($needle in @(
        'X-Console-Product-Code',
        'X-Console-Client-Type',
        'X-CSRF-Token',
        'console_csrf',
        'productCode: "console"',
        'clientType: "pc_web"'
    )) {
    Assert-NotContains $clientText $needle "web API client must not hardcode runtime setting '$needle'"
}

Assert-Contains $typesText "default_client_type: string;" "PublicSettings auth contract must expose default_client_type"
Assert-Contains $systemDomainText "pub default_client_type: String" "backend PublicAuthSettings must expose default_client_type"
Assert-Contains $systemServiceText "default_client_type: self.settings.auth.context.default_client_type.clone()" "public_settings must derive default_client_type from config"
Assert-Contains $routeRegistryText '("default_client_type", string_schema())' "OpenAPI schema must expose default_client_type"
Assert-Contains $routeRegistryText '"default_client_type"' "OpenAPI schema must require default_client_type"
Assert-Contains $clientText "clientType: settings.auth.default_client_type" "web API client must use default_client_type from public settings"

$webSourceRoot = Join-Path $root "web/app/src"
$webFiles = Get-ChildItem -LiteralPath $webSourceRoot -Recurse -File |
    Where-Object { $_.Extension -in @(".ts", ".tsx") } |
    Where-Object { (To-RelativePath $_.FullName) -ne "web/app/src/lib/api/endpoints.ts" }

foreach ($file in $webFiles) {
    $text = Read-Text $file.FullName
    if ($text -like "*/api/v1*") {
        Add-Failure "web runtime source must centralize API paths in lib/api/endpoints.ts: $(To-RelativePath $file.FullName)"
    }
}

if ($failures.Count -gt 0) {
    Write-Host "runtime config boundary scan failed:"
    foreach ($failure in $failures) {
        Write-Host " - $failure"
    }
    exit 1
}

Write-Host "runtime config boundary scan passed: frontend runtime settings come from public settings and API paths stay centralized."
