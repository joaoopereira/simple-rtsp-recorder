# Simple RTSP Recorder - installer / upgrader
#
# Run in PowerShell as Administrator:
#   irm https://raw.githubusercontent.com/joaoopereira/simple-rtsp-recorder/development/install.ps1 | iex
#
# - Upgrading (including from 1.x): installs into the folder of the existing
#   service, keeping settings (prod.env is converted to config.json) and recordings.
# - New install: C:\SimpleRTSPRecorder, or the folder in $env:SRR_INSTALL_DIR.
#
# After this, the app updates itself; this script is only needed once per PC.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Repo = 'joaoopereira/simple-rtsp-recorder'
$ServiceName = 'SimpleRTSPRecorder'
$ExeName = 'simple-rtsp-recorder.exe'
$ZipName = 'simple-rtsp-recorder-win-x64.zip'

function Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }

$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Execute o PowerShell como Administrador e tente novamente.'
}

# Where to install: next to the existing service, if any.
$existing = Get-CimInstance Win32_Service -Filter "Name='$ServiceName'" -ErrorAction SilentlyContinue
if ($existing) {
    $path = $existing.PathName.Trim()
    if ($path.StartsWith('"')) { $path = $path.Substring(1, $path.IndexOf('"', 1) - 1) }
    else { $path = ($path -split '\.exe', 2)[0] + '.exe' }
    $InstallDir = Split-Path $path -Parent
} elseif ($env:SRR_INSTALL_DIR) {
    $InstallDir = $env:SRR_INSTALL_DIR
} else {
    $InstallDir = 'C:\SimpleRTSPRecorder'
}
Step "Pasta de instalação: $InstallDir"

Step 'A obter a versão mais recente...'
$release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
$asset = $release.assets | Where-Object { $_.name -eq $ZipName } | Select-Object -First 1
if (-not $asset) { throw "A versão $($release.tag_name) não tem o ficheiro $ZipName." }
Write-Host "    $($release.tag_name)"

$tmp = Join-Path $env:TEMP ("srr-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Step 'A descarregar...'
    $zip = Join-Path $tmp $ZipName
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip -UseBasicParsing
    Expand-Archive -Path $zip -DestinationPath (Join-Path $tmp 'pkg')

    if ($existing -and $existing.State -ne 'Stopped') {
        Step 'A parar o serviço atual...'
        Stop-Service -Name $ServiceName -Force
        (Get-Service -Name $ServiceName).WaitForStatus('Stopped', [TimeSpan]::FromSeconds(30))
    }
    # The 1.x wrapper may leave the old process behind for a moment.
    Get-Process -Name 'simple-rtsp-recorder' -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep -Seconds 2

    Step 'A copiar ficheiros...'
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $tmp 'pkg\*') -Destination $InstallDir -Recurse -Force

    Step 'A instalar o serviço...'
    & (Join-Path $InstallDir $ExeName) install
    if ($LASTEXITCODE -ne 0) { throw 'Falha ao instalar o serviço (ver mensagem acima).' }
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host ''
Write-Host 'Instalação concluída.' -ForegroundColor Green
Write-Host "Interface: http://localhost:8080   (definições da câmara no ícone da engrenagem)"
Write-Host "Pasta:     $InstallDir"
Write-Host 'As próximas versões são instaladas automaticamente quando não houver gravações.'
