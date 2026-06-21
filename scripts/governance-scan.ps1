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
    $fullPath = (Resolve-Path -LiteralPath $Path).Path
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

function Get-Matches {
    param(
        [string]$Text,
        [string]$Pattern
    )
    return [regex]::Matches($Text, $Pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)
}

function Assert-NoPatternInFiles {
    param(
        [string[]]$Paths,
        [string]$Pattern,
        [string]$Reason
    )
    foreach ($path in $Paths) {
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            continue
        }
        $text = Read-Text $path
        $matches = Get-Matches $text $Pattern
        foreach ($match in $matches) {
            Add-Failure "$Reason in $(To-RelativePath $path): $($match.Value)"
        }
    }
}

$forbiddenOldIdentity = "(Aoi Admin|go-scaffold|go-admin|github\.com/rei0721/go-scaffold|aoi-admin)"
$forbiddenRuntimePlugin = "(/api/v1/plugins|/plugin-api/v1|plugin_registry|plugin_instance|plugin_transport|plugin_api|plugins?:\s|configs\.plugins|remote plugin|remote-plugin)"
$forbiddenEngineeringName = "(?i)(^aoi$|^aoi[-_].+|^aoi[A-Z].*)"

$cargoFiles = Get-ChildItem -LiteralPath $root -Filter "Cargo.toml" -Recurse -File |
    Where-Object { (To-RelativePath $_.FullName) -notmatch "^(aoi-admin|target|\.git)/" }

foreach ($cargoFile in $cargoFiles) {
    $text = Read-Text $cargoFile.FullName
    foreach ($match in [regex]::Matches($text, '(?m)^\s*name\s*=\s*"([^"]+)"')) {
        $name = $match.Groups[1].Value
        if ($name -match $forbiddenEngineeringName -or $name -match "(?i)(go-scaffold|go-admin|aoi-admin)") {
            Add-Failure "forbidden Cargo package name in $(To-RelativePath $cargoFile.FullName): $name"
        }
    }
}

$crateRoot = Join-Path $root "crates"
if (Test-Path -LiteralPath $crateRoot -PathType Container) {
    Get-ChildItem -LiteralPath $crateRoot -Recurse -Directory |
        Where-Object { (To-RelativePath $_.FullName) -notmatch "^(target|\.git)/" } |
        ForEach-Object {
            $segment = $_.Name
            if ($segment -match $forbiddenEngineeringName -or $segment -match "(?i)(go-scaffold|go-admin|aoi-admin)") {
                Add-Failure "forbidden crate path segment: $(To-RelativePath $_.FullName)"
            }
        }
}

$rustFiles = Get-ChildItem -LiteralPath (Join-Path $root "crates") -Filter "*.rs" -Recurse -File
$rustDeclarationPattern = '\b(?:struct|enum|trait|type|fn|mod|const|static)\s+(Aoi[A-Za-z0-9_]*|aoi_[A-Za-z0-9_]*|aoi[A-Z][A-Za-z0-9_]*)\b'
foreach ($rustFile in $rustFiles) {
    $text = Read-Text $rustFile.FullName
    foreach ($match in [regex]::Matches($text, $rustDeclarationPattern)) {
        Add-Failure "forbidden Rust declaration name in $(To-RelativePath $rustFile.FullName): $($match.Groups[1].Value)"
    }
}

$packageJson = Join-Path $root "web/app/package.json"
if (Test-Path -LiteralPath $packageJson -PathType Leaf) {
    $package = Get-Content -LiteralPath $packageJson -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($package.name -match $forbiddenEngineeringName -or $package.name -match "(?i)(go-scaffold|go-admin|aoi-admin)") {
        Add-Failure "forbidden frontend package name in web/app/package.json: $($package.name)"
    }
}

$contractFiles = @(
    (Join-Path $root "docs/api/openapi.yaml"),
    (Join-Path $root "configs/console.example.yaml"),
    (Join-Path $root "configs/console.production.example.yaml"),
    (Join-Path $root "configs/console.secrets.example.yaml"),
    (Join-Path $root ".env.example")
)
Assert-NoPatternInFiles -Paths $contractFiles -Pattern $forbiddenOldIdentity -Reason "forbidden old identity in runtime/config contract"
Assert-NoPatternInFiles -Paths $contractFiles -Pattern $forbiddenRuntimePlugin -Reason "forbidden plugin runtime contract"

$migrationFiles = Get-ChildItem -LiteralPath (Join-Path $root "migrations") -Filter "*.sql" -Recurse -File |
    ForEach-Object { $_.FullName }
Assert-NoPatternInFiles -Paths $migrationFiles -Pattern $forbiddenRuntimePlugin -Reason "forbidden plugin schema"
Assert-NoPatternInFiles -Paths $migrationFiles -Pattern $forbiddenOldIdentity -Reason "forbidden old identity in schema"

$webSourceFiles = Get-ChildItem -LiteralPath (Join-Path $root "web/app/src") -Recurse -File |
    Where-Object { $_.Extension -in @(".ts", ".tsx", ".css", ".json") } |
    ForEach-Object { $_.FullName }
Assert-NoPatternInFiles -Paths $webSourceFiles -Pattern $forbiddenOldIdentity -Reason "forbidden old identity in WebUI runtime source"
Assert-NoPatternInFiles -Paths $webSourceFiles -Pattern "(/api/v1/plugins|/plugin-api/v1|plugins?\.ts|pluginApi|PluginPage)" -Reason "forbidden plugin WebUI runtime source"

if ($failures.Count -gt 0) {
    Write-Host "governance scan failed:"
    foreach ($failure in $failures) {
        Write-Host " - $failure"
    }
    exit 1
}

Write-Host "governance scan passed: neutral engineering names, old identities, plugin runtime contracts, and new schema boundaries are clean."
