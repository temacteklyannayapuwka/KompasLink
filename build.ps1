param([ValidateSet("amd64", "386")][string]$Architecture = "amd64")

$ErrorActionPreference = "Stop"
$previousGoEnvironment = @{
    GOOS = $env:GOOS
    GOARCH = $env:GOARCH
    CGO_ENABLED = $env:CGO_ENABLED
}
Push-Location $PSScriptRoot
try {
    $env:GOOS = "windows"
    $env:GOARCH = $Architecture
    $env:CGO_ENABLED = "0"
    go mod download
    if ($LASTEXITCODE -ne 0) { throw "Dependency download failed" }
    New-Item -ItemType Directory -Force bin | Out-Null
    go build -o bin/kompaslink-diag.exe ./cmd/kompaslink-diag
    if ($LASTEXITCODE -ne 0) { throw "Diagnostic build failed" }
    go build -ldflags="-H windowsgui" -o bin/kompaslink.exe ./cmd/kompaslink
    if ($LASTEXITCODE -ne 0) { throw "GUI build failed" }
    Copy-Item cmd/kompaslink/app.manifest bin/kompaslink.exe.manifest -Force
    Write-Host "Ready: bin/kompaslink.exe"
} finally {
    foreach ($name in $previousGoEnvironment.Keys) {
        if ($null -eq $previousGoEnvironment[$name]) {
            Remove-Item "Env:$name" -ErrorAction SilentlyContinue
        } else {
            Set-Item "Env:$name" $previousGoEnvironment[$name]
        }
    }
    Pop-Location
}
