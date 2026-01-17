# Simple RTSP Recorder - Quick Installer
# Run in PowerShell as Administrator:
# irm https://raw.githubusercontent.com/joaoopereira/simple-rtsp-recorder/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host " Simple RTSP Recorder - Quick Installer" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Check for admin privileges
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
$isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "ERROR: This script requires Administrator privileges." -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again." -ForegroundColor Yellow
    exit 1
}

# Configuration
$InstallDir = "C:\Program Files\SimpleRTSPRecorder"
$TempDir = Join-Path $env:TEMP "SimpleRTSPRecorder"
$GitHubRepo = "joaoopereira/simple-rtsp-recorder"

Write-Host "Installation directory: $InstallDir" -ForegroundColor White
Write-Host ""

# Create temp directory
if (Test-Path $TempDir) {
    Remove-Item $TempDir -Recurse -Force
}
New-Item -ItemType Directory -Path $TempDir | Out-Null

# Get latest release
Write-Host "Fetching latest release..." -ForegroundColor Yellow
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$GitHubRepo/releases/latest"
    $version = $release.tag_name
    $downloadUrl = $release.assets | Where-Object { $_.name -like "*win-x64.zip" } | Select-Object -ExpandProperty browser_download_url
    
    if (-not $downloadUrl) {
        throw "No Windows binary found in latest release"
    }
    
    Write-Host "Latest version: $version" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Failed to fetch latest release" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}

# Download release
Write-Host "Downloading Simple RTSP Recorder..." -ForegroundColor Yellow
$zipFile = Join-Path $TempDir "simple-rtsp-recorder.zip"
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipFile -UseBasicParsing
    Write-Host "Download complete" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Failed to download release" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}

# Extract files
Write-Host "Extracting files..." -ForegroundColor Yellow
try {
    if (Test-Path $InstallDir) {
        Write-Host "Removing existing installation..." -ForegroundColor Yellow
        # Try to stop service if it exists
        $service = Get-Service -Name "SimpleRTSPRecorder" -ErrorAction SilentlyContinue
        if ($service) {
            Write-Host "Stopping existing service..." -ForegroundColor Yellow
            & "$InstallDir\Uninstall-Service.ps1"
        }
        Remove-Item $InstallDir -Recurse -Force
    }
    
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
    Expand-Archive -Path $zipFile -DestinationPath $InstallDir -Force
    Write-Host "Files extracted successfully" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Failed to extract files" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}

# Create environment file
Write-Host ""
Write-Host "Creating default configuration..." -ForegroundColor Yellow
$envFile = Join-Path $InstallDir "prod.env"
if (-not (Test-Path $envFile)) {
    @"
# Simple RTSP Recorder Configuration
# Edit these values according to your RTSP camera settings

# RTSP Camera Settings
RTSP_USER=admin
RTSP_PASSWORD=password
RTSP_IP=192.168.1.100
RTSP_PORT=554
RTSP_SDP=live1.sdp

# Server Settings
PORT=8080

# Recording Settings
OUTPUT_DIR=recordings
"@ | Out-File -FilePath $envFile -Encoding UTF8
    Write-Host "Default configuration created at: $envFile" -ForegroundColor Green
    Write-Host "IMPORTANT: Edit this file with your camera settings!" -ForegroundColor Yellow
}

# Install as Windows Service
Write-Host ""
Write-Host "Installing Windows service..." -ForegroundColor Yellow
try {
    Set-Location $InstallDir
    & "$InstallDir\Install-Service.ps1"
    Write-Host "Service installed successfully" -ForegroundColor Green
} catch {
    Write-Host "WARNING: Service installation failed" -ForegroundColor Yellow
    Write-Host "You can run Install-Service.ps1 manually from: $InstallDir" -ForegroundColor Yellow
}

# Cleanup
Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "================================================" -ForegroundColor Green
Write-Host " Installation Complete!" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "1. Edit the configuration file: $envFile" -ForegroundColor White
Write-Host "2. Start the service or run Install-Service.ps1 again" -ForegroundColor White
Write-Host "3. Open your browser to http://localhost:8080" -ForegroundColor White
Write-Host ""
Write-Host "Installation directory: $InstallDir" -ForegroundColor White
Write-Host ""
