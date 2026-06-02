param(
  [string]$Version = $env:DOCX_VERSION,
  [string]$InstallDir = $env:DOCX_INSTALL_DIR,
  [string]$Repo = $env:DOCX_REPO
)

$ErrorActionPreference = "Stop"

if (-not $Version) { $Version = "latest" }
if (-not $InstallDir) { $InstallDir = Join-Path $env:LOCALAPPDATA "docx\bin" }
if (-not $Repo) { $Repo = "cheng-zuguang/docx" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$asset = "docx_windows_$arch.zip"
if ($Version -eq "latest") {
  $url = "https://github.com/$Repo/releases/latest/download/$asset"
} else {
  $url = "https://github.com/$Repo/releases/download/$Version/$asset"
}

$tmp = New-Item -ItemType Directory -Path (Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString()))
try {
  $archive = Join-Path $tmp.FullName $asset
  Write-Host "Downloading $url"
  Invoke-WebRequest -Uri $url -OutFile $archive
  Expand-Archive -Path $archive -DestinationPath $tmp.FullName -Force
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  Copy-Item -Path (Join-Path $tmp.FullName "docx.exe") -Destination (Join-Path $InstallDir "docx.exe") -Force
  Write-Host "Installed docx to $(Join-Path $InstallDir "docx.exe")"
  Write-Host "Add $InstallDir to PATH if it is not already available."
} finally {
  Remove-Item -Path $tmp.FullName -Recurse -Force
}
