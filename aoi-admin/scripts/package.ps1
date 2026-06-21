param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$PackageArgs
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$Python = if (Get-Command python -ErrorAction SilentlyContinue) { "python" } elseif (Get-Command py -ErrorAction SilentlyContinue) { "py" } else { throw "python was not found on PATH" }

Push-Location $RepoRoot
try {
    & $Python (Join-Path $RepoRoot "scripts/package.py") @PackageArgs
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
