param(
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
Push-Location $Root
try {
    Write-Host "Checking plugin example imports..."
    $exampleImports = rg -n "github\.com/rei0721/go-scaffold/(plugins|_examples)" cmd internal pkg types --glob "*.go" --glob "!**/*_test.go" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $exampleImports
        throw "production code imports plugin examples"
    }

    Write-Host "Checking remote plugin examples do not depend on host Go module..."
    $remoteHostDeps = rg -n "github\.com/rei0721/go-scaffold" _examples/remote-plugins --glob "*.go" --glob "go.mod" --glob "go.sum" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $remoteHostDeps
        throw "remote plugin examples depend on host Go module"
    }

    Write-Host "Checking protocol docs do not present internal Go model as external SDK..."
    $protocolDocDrift = rg -n "(只依赖|外部依赖|远程插件.*依赖).*pkg/plugin/protocol|pkg/plugin/protocol.*(外部依赖|远程插件.*依赖)" docs _examples --glob "*.md" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $protocolDocDrift
        throw "docs describe pkg/plugin/protocol as an external plugin dependency"
    }

    Write-Host "Checking pkg/plugin boundaries..."
    $internalImports = rg -n "github\.com/rei0721/go-scaffold/internal/" pkg/plugin --glob "*.go" --glob "!**/*_test.go" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $internalImports
        throw "pkg/plugin imports internal"
    }
    $rpcImports = rg -n "github\.com/rei0721/go-scaffold/pkg/rpcserver" pkg/plugin --glob "*.go" --glob "!**/*_test.go" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $rpcImports
        throw "pkg/plugin imports rpcserver"
    }

    Write-Host "Checking legacy plugin packages..."
    foreach ($path in @("internal/pluginhost", "internal/modules/plugins")) {
        if (Test-Path $path) {
            throw "legacy plugin package still exists: $path"
        }
    }

    Write-Host "Checking RPC plugin method registration..."
    $rpcPluginMethods = rg -n "plugin\.(negotiate|register|heartbeat|renewLease|unregister|healthCheck|listCapabilities|invoke|pushEvent|subscribeEvent|injectContext|getInjectedSchema|reportStatus|syncMetadata|drain)" cmd internal pkg types --glob "*.go" --glob "!**/*_test.go" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $filtered = $rpcPluginMethods | Where-Object {
            $_ -notmatch "^pkg[\\/]+plugin[\\/]+transport[\\/]+rpc[\\/]+"
        }
        if ($filtered) {
            $filtered
            throw "plugin RPC method leaked outside pkg/plugin/transport/rpc"
        }
    }

    Write-Host "Checking plugin core is transport-neutral..."
    $coreTransportImports = rg -n "github\.com/rei0721/go-scaffold/pkg/plugin/transport|github\.com/rei0721/go-scaffold/pkg/rpcserver|`"net/http`"|`"net/rpc`"" pkg/plugin --glob "*.go" --glob "!**/transport/**" --glob "!**/*_test.go" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $coreTransportImports
        throw "pkg/plugin core must not import concrete transport implementations"
    }

    Write-Host "Checking business modules do not bypass internal/plugin..."
    $businessTransportImports = rg -n "github\.com/rei0721/go-scaffold/pkg/plugin/(transport|router|registry|event|security)" internal/modules --glob "*.go" --glob "!**/*_test.go" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $businessTransportImports
        throw "business modules must call remote plugin capabilities through internal/plugin"
    }

    Write-Host "Checking host config for plugin private config keys..."
    $privateConfig = rg -n "(plugin_id|capabilities|permissions|hooks|endpoint|manifest|private_config)" configs deploy --glob "*.yaml" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $filtered = $privateConfig | Where-Object {
            $_ -notmatch "configs[\\/]+examples[\\/]+plugins-remote-rpc\.example\.yaml" -and
            $_ -notmatch "config\.example\.yaml" -and
            $_ -notmatch "config\.production\.example\.yaml"
        }
        if ($filtered) {
            $filtered
            throw "possible plugin private config in host config"
        }
    }

    Write-Host "Checking config loader does not scan plugin/example/doc directories..."
    $implicitConfigScanning = rg -n "WalkDir|filepath\.Walk|ReadDir|Glob" internal/config pkg/configloader --glob "*.go" --glob "!**/*_test.go" 2>$null
    if ($LASTEXITCODE -eq 0) {
        $implicitConfigScanning
        throw "config loader must use explicit config sources instead of directory scanning"
    }

    Write-Host "Checking plugin metadata keeps protocol and transport separate..."
    $commonSchema = Get-Content -Raw -Encoding UTF8 docs/api/plugin-protocol/schemas/common.schema.json
    if ($commonSchema -notmatch '"transport"' -or $commonSchema -notmatch '"protocol"') {
        throw "plugin protocol schema must expose both protocol and transport"
    }

    Write-Host "Checking plugin observability abstraction..."
    if (-not (Test-Path "pkg/plugin/observability/observability.go")) {
        throw "pkg/plugin/observability package is required for plugin audit/metrics abstraction"
    }
    $hostCore = Get-Content -Raw -Encoding UTF8 pkg/plugin/host.go
    if ($hostCore -notmatch "observability\.Recorder" -or $hostCore -notmatch "recordOperation") {
        throw "plugin host must keep observability recorder integration"
    }

    Write-Host "Checking Go package graph..."
    go list ./... | Out-Null

    if (-not $SkipTests) {
        Write-Host "Running import boundary tests..."
        go test ./internal ./pkg/plugin/... -count=1 -mod=readonly
    }

    Write-Host "Architecture checks passed."
}
finally {
    Pop-Location
}
