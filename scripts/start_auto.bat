@echo off
setlocal enabledelayedexpansion

echo ========================================
echo Neo Framework Auto Start
echo ========================================
echo.

:: 设置默认端口
set HTTP_PORT=28080
set IPC_PORT=29999

:: 自动清理端口函数
:CLEANUP_PORTS
echo Checking ports availability...

:: 检查HTTP端口
set HTTP_OCCUPIED=0
for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%HTTP_PORT% ^| findstr LISTENING') do (
    set HTTP_PID=%%a
    set HTTP_OCCUPIED=1
)

:: 检查IPC端口
set IPC_OCCUPIED=0
for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%IPC_PORT% ^| findstr LISTENING') do (
    set IPC_PID=%%a
    set IPC_OCCUPIED=1
)

:: 如果端口被占用，询问用户
if !HTTP_OCCUPIED! equ 1 (
    echo.
    echo [WARNING] HTTP port %HTTP_PORT% is occupied by process !HTTP_PID!
)
if !IPC_OCCUPIED! equ 1 (
    echo [WARNING] IPC port %IPC_PORT% is occupied by process !IPC_PID!
)

if !HTTP_OCCUPIED! equ 1 if !IPC_OCCUPIED! equ 1 goto :BOTH_OCCUPIED
if !HTTP_OCCUPIED! equ 1 goto :HTTP_ONLY_OCCUPIED
if !IPC_OCCUPIED! equ 1 goto :IPC_ONLY_OCCUPIED
goto :START_NEO

:BOTH_OCCUPIED
echo.
echo Both ports are occupied. Options:
echo   1. Auto-kill both processes
echo   2. Manual selection
echo   3. Cancel
echo.
set /p choice="Select option (1-3): "
if "%choice%"=="1" goto :AUTO_KILL_BOTH
if "%choice%"=="2" goto :MANUAL_MODE
goto :EXIT

:HTTP_ONLY_OCCUPIED
echo.
echo HTTP port is occupied. Options:
echo   1. Auto-kill process
echo   2. Use alternative port
echo   3. Cancel
echo.
set /p choice="Select option (1-3): "
if "%choice%"=="1" goto :KILL_HTTP
if "%choice%"=="2" (
    set /p HTTP_PORT="Enter alternative HTTP port: "
    goto :CLEANUP_PORTS
)
goto :EXIT

:IPC_ONLY_OCCUPIED
echo.
echo IPC port is occupied. Options:
echo   1. Auto-kill process
echo   2. Use alternative port
echo   3. Cancel
echo.
set /p choice="Select option (1-3): "
if "%choice%"=="1" goto :KILL_IPC
if "%choice%"=="2" (
    set /p IPC_PORT="Enter alternative IPC port: "
    goto :CLEANUP_PORTS
)
goto :EXIT

:AUTO_KILL_BOTH
:KILL_HTTP
if defined HTTP_PID (
    echo Terminating HTTP port process !HTTP_PID!...
    taskkill /F /PID !HTTP_PID! >nul 2>&1
    if !errorlevel! neq 0 (
        echo [ERROR] Failed to terminate process. Please run as Administrator.
        pause
        exit /b 1
    )
    echo [SUCCESS] HTTP port process terminated.
)

:KILL_IPC
if defined IPC_PID (
    echo Terminating IPC port process !IPC_PID!...
    taskkill /F /PID !IPC_PID! >nul 2>&1
    if !errorlevel! neq 0 (
        echo [ERROR] Failed to terminate process. Please run as Administrator.
        pause
        exit /b 1
    )
    echo [SUCCESS] IPC port process terminated.
)

:: 等待端口释放
echo Waiting for ports to be released...
timeout /t 2 /nobreak >nul
goto :START_NEO

:MANUAL_MODE
echo.
echo Manual port selection:
set /p HTTP_PORT="Enter HTTP port (default 28080): "
set /p IPC_PORT="Enter IPC port (default 29999): "
goto :CLEANUP_PORTS

:START_NEO
echo.
echo ========================================
echo Starting Neo Framework
echo ========================================
echo HTTP Gateway: http://localhost:%HTTP_PORT%
echo IPC Server: localhost:%IPC_PORT%
echo ========================================
echo.

:: 保存端口配置
echo NEO_HTTP_PORT=%HTTP_PORT% > %TEMP%\neo_ports.env
echo NEO_IPC_PORT=%IPC_PORT% >> %TEMP%\neo_ports.env

:: 启动Neo Framework
go run cmd/neo/main.go -http :%HTTP_PORT% -ipc :%IPC_PORT%

goto :END

:EXIT
echo.
echo Operation cancelled.

:END
endlocal
pause