param([ValidateSet("amd64", "386")][string]$Architecture = "amd64")

$ErrorActionPreference = "Stop"
$previousGoEnvironment = @{
    GOOS = $env:GOOS
    GOARCH = $env:GOARCH
    CGO_ENABLED = $env:CGO_ENABLED
    GOTOOLCHAIN = $env:GOTOOLCHAIN
    GOENV = $env:GOENV
    GOWORK = $env:GOWORK
    GOFLAGS = $env:GOFLAGS
    GOEXPERIMENT = $env:GOEXPERIMENT
    GOAMD64 = $env:GOAMD64
    GO386 = $env:GO386
}
Push-Location $PSScriptRoot
try {
    $env:GOTOOLCHAIN = "local"
    $env:GOENV = "off"
    $env:GOWORK = "off"
    $env:GOFLAGS = ""
    $env:GOEXPERIMENT = ""
    $env:GOAMD64 = "v1"
    $env:GO386 = "sse2"
    $requiredVersion = (Get-Content .go-version -Raw).Trim()
    $actualVersion = go env GOVERSION
    if ($LASTEXITCODE -ne 0) { throw "Cannot determine Go version" }
    if ($actualVersion -ne "go$requiredVersion") { throw "Install Go $requiredVersion; found $actualVersion" }
    $env:GOOS = (go env GOHOSTOS).Trim()
    if ($LASTEXITCODE -ne 0) { throw "Cannot determine build host OS" }
    $env:GOARCH = (go env GOHOSTARCH).Trim()
    if ($LASTEXITCODE -ne 0) { throw "Cannot determine build host architecture" }
    $env:CGO_ENABLED = "0"
    go run github.com/akavel/rsrc@v0.10.2 -manifest cmd/kompaslink/app.manifest -arch $Architecture -o "cmd/kompaslink/manifest_windows_$Architecture.syso"
    if ($LASTEXITCODE -ne 0) { throw "Manifest resource generation failed" }
    $env:GOOS = "windows"
    $env:GOARCH = $Architecture
    $env:CGO_ENABLED = "0"
    go mod download
    if ($LASTEXITCODE -ne 0) { throw "Dependency download failed" }
    go mod verify
    if ($LASTEXITCODE -ne 0) { throw "Dependency verification failed" }
    New-Item -ItemType Directory -Force bin | Out-Null
    go build -trimpath -buildvcs=false -mod=readonly -o bin/kompaslink-diag.exe ./cmd/kompaslink-diag
    if ($LASTEXITCODE -ne 0) { throw "Diagnostic build failed" }
    go build -trimpath -buildvcs=false -mod=readonly -ldflags="-H windowsgui" -o bin/kompaslink.exe ./cmd/kompaslink
    if ($LASTEXITCODE -ne 0) { throw "GUI build failed" }
    $files = @("kompaslink.exe", "kompaslink-diag.exe")
    $checksums = foreach ($file in $files) {
        if (-not (Test-Path "bin/$file" -PathType Leaf)) { throw "Missing build output: $file. Check endpoint protection reports; do not disable protection." }
        $hash = Get-FileHash "bin/$file" -Algorithm SHA256
        "$($hash.Hash.ToLowerInvariant())  $file"
    }
    $checksums | Set-Content bin/SHA256SUMS.txt -Encoding ascii
    $metadata = go version -m bin/kompaslink.exe bin/kompaslink-diag.exe
    if ($LASTEXITCODE -ne 0) { throw "Cannot read build metadata" }
    $metadata | Set-Content bin/build-info.txt -Encoding utf8
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
