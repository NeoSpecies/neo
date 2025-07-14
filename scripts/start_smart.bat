@echo off
setlocal enabledelayedexpansion

echo ========================================
echo Neo Framework Smart Launcher
echo ========================================
echo.

:: 设置默认端口
set HTTP_PORT=28080
set IPC_PORT=29999

:: 检查HTTP端口
echo Checking HTTP port %HTTP_PORT%...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%HTTP_PORT% ^| findstr LISTENING') do (
    set PID=%%a
    goto :HTTP_OCCUPIED
)
goto :CHECK_IPC

:HTTP_OCCUPIED
echo.
echo [WARNING] Port %HTTP_PORT% is already in use by process !PID!
echo.
echo Process details:
wmic process where ProcessId=!PID! get ProcessId,Name,CommandLine 2>nul | findstr /v "^$"
echo.
echo Options:
echo   1. Kill the process and continue
echo   2. Use alternative port
echo   3. Cancel
echo.
set /p choice="Select option (1-3): "

if "%choice%"=="1" (
    echo Terminating process !PID!...
    taskkill /F /PID !PID! >nul 2>&1
    if !errorlevel! equ 0 (
        echo Process terminated successfully.
        timeout /t 2 /nobreak >nul
    ) else (
        echo Failed to terminate process. Please run as Administrator.
        pause
        exit /b 1
    )
) else if "%choice%"=="2" (
    set /p HTTP_PORT="Enter alternative HTTP port: "
) else (
    echo Operation cancelled.
    pause
    exit /b 0
)

:CHECK_IPC
:: 检查IPC端口
echo Checking IPC port %IPC_PORT%...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%IPC_PORT% ^| findstr LISTENING') do (
    set PID=%%a
    goto :IPC_OCCUPIED
)
goto :START_NEO

:IPC_OCCUPIED
echo.
echo [WARNING] Port %IPC_PORT% is already in use by process !PID!
echo.
echo Process details:
wmic process where ProcessId=!PID! get ProcessId,Name,CommandLine 2>nul | findstr /v "^$"
echo.
echo Options:
echo   1. Kill the process and continue
echo   2. Use alternative port
echo   3. Cancel
echo.
set /p choice="Select option (1-3): "

if "%choice%"=="1" (
    echo Terminating process !PID!...
    taskkill /F /PID !PID! >nul 2>&1
    if !errorlevel! equ 0 (
        echo Process terminated successfully.
        timeout /t 2 /nobreak >nul
    ) else (
        echo Failed to terminate process. Please run as Administrator.
        pause
        exit /b 1
    )
) else if "%choice%"=="2" (
    set /p IPC_PORT="Enter alternative IPC port: "
) else (
    echo Operation cancelled.
    pause
    exit /b 0
)

:START_NEO
echo.
echo ========================================
echo Starting Neo Framework
echo ========================================
echo HTTP Gateway: http://localhost:%HTTP_PORT%
echo IPC Server: localhost:%IPC_PORT%
echo ========================================
echo.

:: 保存端口配置到临时文件，供Python服务使用
echo HTTP_PORT=%HTTP_PORT% > neo_ports.env
echo IPC_PORT=%IPC_PORT% >> neo_ports.env

:: 启动Neo Framework
go run cmd/neo/main.go -http :%HTTP_PORT% -ipc :%IPC_PORT%

:: 清理临时文件
del neo_ports.env 2>nul

endlocal