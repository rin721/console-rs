param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$failures = New-Object System.Collections.Generic.List[string]

$rawTokenPattern = "session_token_|refresh_token_|api_token_|invitation_token_|password_reset_token_|email_verify_token_|mfa_recovery_code_"
$storagePattern = "\b(localStorage|sessionStorage)\b"
$loggingSinkPattern = "(?i)(println!|eprintln!|\b(trace|debug|info|warn|error)!\s*\(|console\.(log|debug|info|warn|error)\s*\()"
$sensitiveLogPattern = "(?i)(token|secret|password|cookie|authorization|credential|mfa|csrf|raw_?token|refresh)"
$pendingUrlPattern = "(?i)(invite|invitation|reset|verify|verification|token|password)"
$textExtensions = @(
    ".css",
    ".env",
    ".example",
    ".html",
    ".js",
    ".json",
    ".md",
    ".mjs",
    ".ps1",
    ".rs",
    ".sql",
    ".toml",
    ".ts",
    ".tsx",
    ".txt",
    ".yaml",
    ".yml"
)
$scanEntries = @(
    ".agents",
    ".env.example",
    "AGENTS.md",
    "Cargo.toml",
    "README.md",
    "configs",
    "crates",
    "docs",
    "migrations",
    "web/app/src",
    "web/app/tests"
)
$artifactEntries = @(
    "web/app/playwright-report",
    "web/app/test-results",
    "web/app/.vite"
)

function Get-RelativePath {
    param([string]$Path)
    $full = [System.IO.Path]::GetFullPath($Path)
    $root = [System.IO.Path]::GetFullPath($rootPath).TrimEnd('\', '/')
    if ($full.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $full.Substring($root.Length).TrimStart('\', '/').Replace('\', '/')
    }
    return $full.Replace('\', '/')
}

function Test-TextFile {
    param([System.IO.FileInfo]$File)
    if ($textExtensions -contains $File.Extension) {
        return $true
    }
    return $File.Name -eq ".gitignore" -or $File.Name -eq ".env.example"
}

function Test-ExcludedPath {
    param([string]$RelativePath)
    return $RelativePath -match "(^|/)(aoi-admin|data|node_modules|target|dist|build|\.git)(/|$)"
}

function Add-Failure {
    param(
        [string]$Path,
        [int]$LineNumber,
        [string]$Rule,
        [string]$Text
    )
    $shortText = $Text.Trim()
    if ($shortText.Length -gt 180) {
        $shortText = $shortText.Substring(0, 180) + "..."
    }
    $failures.Add("${Path}:${LineNumber} [$Rule] $shortText")
}

function Get-ScanFiles {
    param([string[]]$Entries)
    $files = New-Object System.Collections.Generic.List[System.IO.FileInfo]
    foreach ($entry in $Entries) {
        $path = Join-Path $rootPath $entry
        if (-not (Test-Path -LiteralPath $path)) {
            continue
        }
        $item = Get-Item -LiteralPath $path
        if ($item.PSIsContainer) {
            Get-ChildItem -LiteralPath $path -Recurse -File | ForEach-Object {
                $relative = Get-RelativePath $_.FullName
                if (-not (Test-ExcludedPath $relative) -and (Test-TextFile $_)) {
                    $files.Add($_)
                }
            }
        } elseif (Test-TextFile $item) {
            $files.Add($item)
        }
    }
    return $files
}

function Scan-File {
    param(
        [System.IO.FileInfo]$File,
        [bool]$IsArtifact
    )
    $relative = Get-RelativePath $File.FullName
    try {
        $lines = [System.IO.File]::ReadAllLines($File.FullName)
    } catch {
        Add-Failure $relative 0 "read-text" "failed to read file as text: $($_.Exception.Message)"
        return
    }

    for ($index = 0; $index -lt $lines.Length; $index++) {
        $line = $lines[$index]
        $lineNumber = $index + 1

        if ($IsArtifact -and $line -match $rawTokenPattern) {
            Add-Failure $relative $lineNumber "artifact-raw-token" $line
        }

        if ($relative -like "web/app/src/*" -and $line -match $rawTokenPattern) {
            Add-Failure $relative $lineNumber "frontend-runtime-raw-token" $line
        }

        if ($relative -like "web/app/src/*" -and $line -match $storagePattern) {
            if ($line -notmatch "console-locale") {
                Add-Failure $relative $lineNumber "frontend-storage-sensitive" $line
            }
        }

        if ($relative -like "web/app/tests/*" -and $line -match $storagePattern -and $line -match $rawTokenPattern) {
            if ($line -notmatch "\.not\.") {
                Add-Failure $relative $lineNumber "test-storage-token-assertion" $line
            }
        }

        if ($relative -like "web/app/src/*" -and $line -match "(URLSearchParams|location\.search|searchParams)" -and $line -match $pendingUrlPattern) {
            Add-Failure $relative $lineNumber "frontend-url-pending-token" $line
        }

        if (($relative -like "crates/*" -or $relative -like "web/app/src/*") -and $line -match $loggingSinkPattern -and $line -match $sensitiveLogPattern) {
            Add-Failure $relative $lineNumber "runtime-sensitive-log" $line
        }
    }
}

Get-ScanFiles $scanEntries | ForEach-Object {
    Scan-File $_ $false
}

Get-ScanFiles $artifactEntries | ForEach-Object {
    Scan-File $_ $true
}

if ($failures.Count -gt 0) {
    [Console]::Out.WriteLine("sensitive scan failed: possible token/secret leak sinks found.")
    foreach ($failure in $failures) {
        [Console]::Out.WriteLine($failure)
    }
    exit 1
} else {
    [Console]::Out.WriteLine("sensitive scan passed: no raw token/secret leak sinks found in runtime, URL, storage, logs, or text artifacts.")
}
