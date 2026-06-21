param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$failures = New-Object System.Collections.Generic.List[string]
$utf8Strict = [System.Text.UTF8Encoding]::new($false, $true)

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
    ".github",
    ".gitignore",
    "AGENTS.md",
    "Cargo.toml",
    "README.md",
    "configs",
    "crates",
    "docs",
    "migrations",
    "scripts",
    "web/app"
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

function Test-ExcludedPath {
    param([string]$RelativePath)
    return $RelativePath -match "(^|/)(aoi-admin|data|node_modules|target|dist|build|playwright-report|test-results|\.git)(/|$)"
}

function Test-TextFile {
    param([System.IO.FileInfo]$File)
    if ($textExtensions -contains $File.Extension) {
        return $true
    }
    return $File.Name -in @(".env.example", ".gitignore")
}

function Add-Failure {
    param(
        [string]$Path,
        [int]$LineNumber
    )
    $failures.Add("${Path}:${LineNumber} trailing whitespace") | Out-Null
}

function Scan-File {
    param([System.IO.FileInfo]$File)
    $relative = Get-RelativePath $File.FullName
    try {
        $lines = [System.IO.File]::ReadAllLines($File.FullName, $utf8Strict)
    } catch {
        $failures.Add("${relative}:0 failed to read as UTF-8 text: $($_.Exception.Message)") | Out-Null
        return
    }

    for ($index = 0; $index -lt $lines.Length; $index++) {
        if ($lines[$index] -match "[ \t]+$") {
            Add-Failure $relative ($index + 1)
        }
    }
}

foreach ($entry in $scanEntries) {
    $path = Join-Path $rootPath $entry
    if (-not (Test-Path -LiteralPath $path)) {
        continue
    }

    $item = Get-Item -LiteralPath $path
    if ($item.PSIsContainer) {
        Get-ChildItem -LiteralPath $path -Recurse -File | ForEach-Object {
            $relative = Get-RelativePath $_.FullName
            if (-not (Test-ExcludedPath $relative) -and (Test-TextFile $_)) {
                Scan-File $_
            }
        }
    } elseif (Test-TextFile $item) {
        Scan-File $item
    }
}

if ($failures.Count -gt 0) {
    [Console]::Out.WriteLine("trailing whitespace scan failed:")
    foreach ($failure in $failures) {
        [Console]::Out.WriteLine(" - $failure")
    }
    exit 1
}

[Console]::Out.WriteLine("trailing whitespace scan passed: no trailing spaces or tabs found in tracked project text areas.")
