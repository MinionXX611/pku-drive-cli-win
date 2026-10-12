[CmdletBinding()]
param(
    [string]$Version = "dev",
    [string]$Commit = "",
    [string]$BuildDate = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Commit)) {
    $Commit = (git rev-parse --short=12 HEAD).Trim()
}
if ([string]::IsNullOrWhiteSpace($BuildDate)) {
    $epoch = [Int64]((git log -1 --format=%ct).Trim())
    $BuildDate = [DateTimeOffset]::FromUnixTimeSeconds($epoch).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
}
if ($Version -notmatch '^[0-9A-Za-z._+-]+$' -or $Commit -notmatch '^[0-9a-f]{7,64}$' -or $BuildDate -notmatch '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$') {
    throw "Invalid build metadata."
}

$go = Get-Command go -ErrorAction Stop
$outputDirectory = Join-Path $PSScriptRoot "..\bin"
New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
$output = Join-Path $outputDirectory "pku-drive.exe"
$ldflags = "-s -w -buildid= -X main.version=$Version -X main.commit=$Commit -X main.buildDate=$BuildDate"

& $go.Source build -trimpath -buildvcs=false -ldflags $ldflags -o $output "./cmd/pku-drive"
if ($LASTEXITCODE -ne 0) { throw "Go build failed." }
& $output version
if ($LASTEXITCODE -ne 0) { throw "Built executable failed its version check." }
