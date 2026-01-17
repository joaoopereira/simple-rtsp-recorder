# Simple RTSP Recorder - Windows Service Uninstaller
# Run as Administrator

$ErrorActionPreference = "Stop"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host " Simple RTSP Recorder - Service Uninstaller" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Check for admin privileges
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
$isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "ERROR: This script requires Administrator privileges." -ForegroundColor Red
    Write-Host "Please right-click and select 'Run as Administrator'" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

# Configuration
$ServiceName = "SimpleRTSPRecorder"
$WinSwService = Join-Path $PSScriptRoot "SimpleRTSPRecorder.exe"

# Check if service exists
$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

if (-not $service) {
    Write-Host "Service '$ServiceName' is not installed." -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 0
}

# Check if WinSW service wrapper exists
if (-not (Test-Path $WinSwService)) {
    Write-Host "ERROR: WinSW service wrapper not found at: $WinSwService" -ForegroundColor Red
    Write-Host "Attempting to remove service using sc.exe..." -ForegroundColor Yellow
    
    if ($service.Status -eq 'Running') {
        Stop-Service -Name $ServiceName -Force
        Start-Sleep -Seconds 2
    }
    
    & sc.exe delete $ServiceName
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Service removed successfully" -ForegroundColor Green
    }
    Read-Host "Press Enter to exit"
    exit 0
}

Write-Host "Stopping and removing service..." -ForegroundColor Cyan

# Stop the service using WinSW
if ($service.Status -eq 'Running') {
    & $WinSwService stop
    Write-Host "Service stopped" -ForegroundColor Green
    Start-Sleep -Seconds 2
}

# Uninstall the service using WinSW
& $WinSwService uninstall

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "================================================" -ForegroundColor Green
    Write-Host " Service uninstalled successfully!" -ForegroundColor Green
    Write-Host "================================================" -ForegroundColor Green
    
    # Clean up service executable
    Start-Sleep -Seconds 1
    if (Test-Path $WinSwService) {
        Remove-Item $WinSwService -Force -ErrorAction SilentlyContinue
        Write-Host "Removed service wrapper" -ForegroundColor Green
    }
} else {
    Write-Host "ERROR: Failed to remove service" -ForegroundColor Red
    Write-Host "You may need to restart your computer and try again." -ForegroundColor Yellow
}

Write-Host ""
Read-Host "Press Enter to exit"
