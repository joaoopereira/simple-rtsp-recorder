# Simple RTSP Recorder - Windows Service Installer
# Run as Administrator

$ErrorActionPreference = "Stop"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host " Simple RTSP Recorder - Service Installer" -ForegroundColor Cyan
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
$ServiceDisplayName = "Simple RTSP Recorder"
$ServiceDescription = "RTSP camera stream recorder with web interface"
$ScriptPath = $PSScriptRoot
$ExePath = Join-Path $ScriptPath "simple-rtsp-recorder.exe"
$WinSwExe = Join-Path $ScriptPath "WinSW.exe"
$WinSwService = Join-Path $ScriptPath "SimpleRTSPRecorder.exe"
$ConfigXml = Join-Path $ScriptPath "SimpleRTSPRecorder.xml"
$EnvFile = Join-Path $ScriptPath "prod.env"
$RecordingsDir = Join-Path $ScriptPath "recordings"
$LogsDir = Join-Path $ScriptPath "logs"

Write-Host "Installation Directory: $ScriptPath" -ForegroundColor White
Write-Host ""

# Check if WinSW exists
if (-not (Test-Path $WinSwExe)) {
    Write-Host "WinSW.exe not found. Downloading..." -ForegroundColor Yellow
    try {
        $WinSwUrl = "https://github.com/winsw/winsw/releases/download/v2.12.0/WinSW-x64.exe"
        Invoke-WebRequest -Uri $WinSwUrl -OutFile $WinSwExe -UseBasicParsing
        Write-Host "Downloaded WinSW successfully" -ForegroundColor Green
    } catch {
        Write-Host "ERROR: Failed to download WinSW" -ForegroundColor Red
        Write-Host "Please download WinSW manually from: https://github.com/winsw/winsw/releases" -ForegroundColor Yellow
        Write-Host "Save it as WinSW.exe in: $ScriptPath" -ForegroundColor Yellow
        Read-Host "Press Enter to exit"
        exit 1
    }
}

# Check if executable exists
if (-not (Test-Path $ExePath)) {
    Write-Host "ERROR: Executable not found at: $ExePath" -ForegroundColor Red
    Write-Host "Please ensure simple-rtsp-recorder.exe is in the same folder as this script." -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

# Check if prod.env exists
if (-not (Test-Path $EnvFile)) {
    Write-Host "WARNING: prod.env not found. Creating from .env..." -ForegroundColor Yellow
    $DefaultEnv = Join-Path $ScriptPath ".env"
    if (Test-Path $DefaultEnv) {
        Copy-Item $DefaultEnv $EnvFile
        Write-Host "Created prod.env from .env" -ForegroundColor Green
    } else {
        Write-Host "ERROR: Neither prod.env nor .env found!" -ForegroundColor Red
        Write-Host "Please create prod.env with your configuration." -ForegroundColor Yellow
        Read-Host "Press Enter to exit"
        exit 1
    }
}

# Create recordings directory if it doesn't exist
if (-not (Test-Path $RecordingsDir)) {
    New-Item -ItemType Directory -Path $RecordingsDir | Out-Null
    Write-Host "Created recordings directory" -ForegroundColor Green
}

# Create logs directory if it doesn't exist
if (-not (Test-Path $LogsDir)) {
    New-Item -ItemType Directory -Path $LogsDir | Out-Null
    Write-Host "Created logs directory" -ForegroundColor Green
}

# Check if XML config exists
if (-not (Test-Path $ConfigXml)) {
    Write-Host "ERROR: SimpleRTSPRecorder.xml configuration file not found!" -ForegroundColor Red
    Write-Host "Expected at: $ConfigXml" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

# Copy WinSW.exe to SimpleRTSPRecorder.exe (WinSW naming convention)
Copy-Item $WinSwExe $WinSwService -Force
Write-Host "Prepared service executable" -ForegroundColor Green

# Check if service already exists
$existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

if ($existingService) {
    Write-Host "Service already exists. Removing old service..." -ForegroundColor Yellow
    
    # Use WinSW to uninstall
    & $WinSwService uninstall
    Start-Sleep -Seconds 2
    Write-Host "Removed existing service" -ForegroundColor Green
}

Write-Host ""
Write-Host "Installing service using WinSW..." -ForegroundColor Cyan

# Install the service using WinSW
try {
    & $WinSwService install
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Service installed successfully" -ForegroundColor Green
        
        # Start the service
        Write-Host "Starting service..." -ForegroundColor Cyan
        & $WinSwService start
        
        if ($LASTEXITCODE -eq 0) {
            Start-Sleep -Seconds 3
            
            $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($service -and $service.Status -eq 'Running') {
                Write-Host ""
                Write-Host "================================================" -ForegroundColor Green
                Write-Host " Service installed and started successfully!" -ForegroundColor Green
                Write-Host "================================================" -ForegroundColor Green
                Write-Host ""
                Write-Host "Service Details:" -ForegroundColor White
                Write-Host "  Name: $ServiceName" -ForegroundColor White
                Write-Host "  Status: Running" -ForegroundColor Green
                Write-Host "  Startup Type: Automatic" -ForegroundColor White
                Write-Host ""
                Write-Host "Web Interface: http://localhost:8080" -ForegroundColor Cyan
                Write-Host ""
                Write-Host "Logs:" -ForegroundColor White
                Write-Host "  Service wrapper: $LogsDir\SimpleRTSPRecorder.wrapper.log" -ForegroundColor White
                Write-Host "  Application: $ScriptPath\server.log" -ForegroundColor White
                Write-Host ""
                Write-Host "To manage the service:" -ForegroundColor White
                Write-Host "  - Stop and uninstall: .\Uninstall-Service.ps1" -ForegroundColor White
                Write-Host "  - Windows Services: services.msc" -ForegroundColor White
            } else {
                Write-Host ""
                Write-Host "WARNING: Service installed but not running." -ForegroundColor Yellow
                Write-Host "Status: $($service.Status)" -ForegroundColor Yellow
                Write-Host ""
                Write-Host "Check the logs:" -ForegroundColor White
                Write-Host "  $LogsDir\SimpleRTSPRecorder.wrapper.log" -ForegroundColor White
                Write-Host "  $ScriptPath\server.log" -ForegroundColor White
            }
        } else {
            Write-Host ""
            Write-Host "ERROR: Service installed but failed to start" -ForegroundColor Red
            Write-Host ""
            Write-Host "Check the logs:" -ForegroundColor White
            Write-Host "  $LogsDir\SimpleRTSPRecorder.wrapper.log" -ForegroundColor White
        }
    } else {
        Write-Host "ERROR: Failed to install service" -ForegroundColor Red
        Write-Host ""
        Write-Host "Check the logs:" -ForegroundColor White
        Write-Host "  $LogsDir\SimpleRTSPRecorder.wrapper.log" -ForegroundColor White
    }
} catch {
    Write-Host "ERROR: Exception during service installation" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
}

Write-Host ""
Read-Host "Press Enter to exit"
