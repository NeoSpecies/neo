# Neo Framework PowerShell Startup Script
param(
    [int]$HttpPort = 28080,
    [int]$IpcPort = 29999,
    [switch]$AutoCleanup,
    [switch]$Force
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Neo Framework PowerShell Launcher" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

function Test-PortInUse {
    param([int]$Port)
    
    $connection = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if ($connection) {
        $process = Get-Process -Id $connection.OwningProcess -ErrorAction SilentlyContinue
        return @{
            InUse = $true
            ProcessId = $connection.OwningProcess
            ProcessName = $process.Name
            ProcessPath = $process.Path
        }
    }
    return @{ InUse = $false }
}

function Stop-PortProcess {
    param([int]$Port)
    
    $portInfo = Test-PortInUse -Port $Port
    if ($portInfo.InUse) {
        Write-Host "Port $Port is used by process: $($portInfo.ProcessName) (PID: $($portInfo.ProcessId))" -ForegroundColor Yellow
        
        if ($Force -or $AutoCleanup) {
            Write-Host "Terminating process..." -ForegroundColor Yellow
            Stop-Process -Id $portInfo.ProcessId -Force
            Start-Sleep -Seconds 2
            Write-Host "Process terminated." -ForegroundColor Green
            return $true
        } else {
            $choice = Read-Host "Terminate this process? (Y/N)"
            if ($choice -eq 'Y' -or $choice -eq 'y') {
                Stop-Process -Id $portInfo.ProcessId -Force
                Start-Sleep -Seconds 2
                Write-Host "Process terminated." -ForegroundColor Green
                return $true
            }
            return $false
        }
    }
    return $true
}

# Check if running as Administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
if (-not $isAdmin -and -not $Force) {
    Write-Host "WARNING: Not running as Administrator. May not be able to terminate some processes." -ForegroundColor Yellow
    Write-Host ""
}

# Check HTTP port
Write-Host "Checking HTTP port $HttpPort..." -NoNewline
$httpPortInfo = Test-PortInUse -Port $HttpPort
if ($httpPortInfo.InUse) {
    Write-Host " OCCUPIED" -ForegroundColor Red
    Write-Host "Process: $($httpPortInfo.ProcessName) (PID: $($httpPortInfo.ProcessId))" -ForegroundColor Yellow
    
    if (-not (Stop-PortProcess -Port $HttpPort)) {
        # Try alternative port
        $newPort = Read-Host "Enter alternative HTTP port (or press Enter to cancel)"
        if ($newPort) {
            $HttpPort = [int]$newPort
        } else {
            Write-Host "Operation cancelled." -ForegroundColor Red
            exit
        }
    }
} else {
    Write-Host " FREE" -ForegroundColor Green
}

# Check IPC port
Write-Host "Checking IPC port $IpcPort..." -NoNewline
$ipcPortInfo = Test-PortInUse -Port $IpcPort
if ($ipcPortInfo.InUse) {
    Write-Host " OCCUPIED" -ForegroundColor Red
    Write-Host "Process: $($ipcPortInfo.ProcessName) (PID: $($ipcPortInfo.ProcessId))" -ForegroundColor Yellow
    
    if (-not (Stop-PortProcess -Port $IpcPort)) {
        # Try alternative port
        $newPort = Read-Host "Enter alternative IPC port (or press Enter to cancel)"
        if ($newPort) {
            $IpcPort = [int]$newPort
        } else {
            Write-Host "Operation cancelled." -ForegroundColor Red
            exit
        }
    }
} else {
    Write-Host " FREE" -ForegroundColor Green
}

# Save port configuration for Python services
$portConfig = @"
NEO_HTTP_PORT=$HttpPort
NEO_IPC_PORT=$IpcPort
"@
$portConfig | Out-File -FilePath "$env:TEMP\neo_ports.env" -Encoding UTF8

# Start Neo Framework
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "Starting Neo Framework" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "HTTP Gateway: http://localhost:$HttpPort" -ForegroundColor Cyan
Write-Host "IPC Server: localhost:$IpcPort" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

# Start the application
$process = Start-Process -FilePath "go" -ArgumentList "run", "cmd/neo/main.go", "-http", ":$HttpPort", "-ipc", ":$IpcPort" -NoNewWindow -PassThru

# Monitor the process
Write-Host "Neo Framework is running. Press Ctrl+C to stop..." -ForegroundColor Yellow
try {
    Wait-Process -Id $process.Id
} catch {
    Write-Host "Stopping Neo Framework..." -ForegroundColor Yellow
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
}

# Cleanup
Remove-Item "$env:TEMP\neo_ports.env" -ErrorAction SilentlyContinue
Write-Host "Neo Framework stopped." -ForegroundColor Red