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
$InstallDir = Join-Path $pwd "SimpleRTSPRecorder"
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
    # Use BITS for better progress display
    Start-BitsTransfer -Source $downloadUrl -Destination $zipFile -Description "Downloading Simple RTSP Recorder" -DisplayName "Simple RTSP Recorder"
    Write-Host "Download complete" -ForegroundColor Green
} catch {
    # Fallback to Invoke-WebRequest if BITS fails
    Write-Host "Trying alternative download method..." -ForegroundColor Yellow
    try {
        $ProgressPreference = 'Continue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipFile -UseBasicParsing
        Write-Host "Download complete" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to download release" -ForegroundColor Red
        Write-Host $_.Exception.Message -ForegroundColor Red
        exit 1
    } finally {
        $ProgressPreference = 'SilentlyContinue'
    }
}

# Extract files
Write-Host "Extracting files..." -ForegroundColor Yellow
try {
    if (Test-Path $InstallDir) {
        Write-Host "Existing installation found. Removing..." -ForegroundColor Yellow
        
        # Uninstall service if it exists
        $service = Get-Service -Name "SimpleRTSPRecorder" -ErrorAction SilentlyContinue
        if ($service) {
            Write-Host "Uninstalling existing service..." -ForegroundColor Yellow
            $uninstallScript = Join-Path $InstallDir "Uninstall-Service.ps1"
            if (Test-Path $uninstallScript) {
                & $uninstallScript
                
                # Wait for service to be fully removed
                Write-Host "Waiting for service to be fully removed..." -ForegroundColor Yellow
                $maxWait = 30
                $waited = 0
                while ((Get-Service -Name "SimpleRTSPRecorder" -ErrorAction SilentlyContinue) -and ($waited -lt $maxWait)) {
                    Start-Sleep -Seconds 1
                    $waited++
                }
                
                if (Get-Service -Name "SimpleRTSPRecorder" -ErrorAction SilentlyContinue) {
                    Write-Host "WARNING: Service still exists after uninstall. Manual intervention may be required." -ForegroundColor Yellow
                } else {
                    Write-Host "Service uninstalled successfully" -ForegroundColor Green
                }
            }
        }
        
        # Remove installation directory
        Remove-Item $InstallDir -Recurse -Force
        Write-Host "Previous installation removed" -ForegroundColor Green
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
Write-Host "Checking configuration file..." -ForegroundColor Yellow
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
} else {
    Write-Host "Existing configuration preserved at: $envFile" -ForegroundColor Green
}

# Install as Windows Service
Write-Host ""
Write-Host "Installing Windows service..." -ForegroundColor Yellow
try {
    $installServiceScript = Join-Path $InstallDir "Install-Service.ps1"
    if (Test-Path $installServiceScript) {
        Push-Location $InstallDir
        & $installServiceScript
        Pop-Location
        Write-Host "Service installed successfully" -ForegroundColor Green
    } else {
        Write-Host "WARNING: Install-Service.ps1 not found" -ForegroundColor Yellow
    }
} catch {
    Write-Host "WARNING: Service installation failed" -ForegroundColor Yellow
    Write-Host "You can run Install-Service.ps1 manually from: $InstallDir" -ForegroundColor Yellow
    Pop-Location -ErrorAction SilentlyContinue
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
