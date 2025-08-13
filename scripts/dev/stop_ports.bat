@echo off
setlocal enabledelayedexpansion

echo ========================================
echo Neo Framework Port Cleanup Tool
echo ========================================
echo.

:: 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] This script requires Administrator privileges.
    echo Please run as Administrator.
    echo.
    pause
    exit /b 1
)

:: 默认端口
set HTTP_PORT=8080
set IPC_PORT=9999

echo Checking and cleaning default Neo Framework ports...
echo.

:: 检查并终止HTTP端口进程
echo Checking HTTP port %HTTP_PORT%...
set HTTP_PID=
for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%HTTP_PORT% ^| findstr LISTENING') do (
    set HTTP_PID=%%a
)

if defined HTTP_PID (
    echo Port %HTTP_PORT% is occupied by process %HTTP_PID%
    
    :: 获取进程信息
    for /f "tokens=2 delims=," %%a in ('wmic process where ProcessId^=%HTTP_PID% get Name /format:csv 2^>nul ^| findstr /v "^$" ^| findstr /v "Node,Name"') do (
        echo Process name: %%a
    )
    
    echo Terminating process %HTTP_PID%...
    taskkill /F /PID %HTTP_PID% >nul 2>&1
    if !errorlevel! equ 0 (
        echo [SUCCESS] Process %HTTP_PID% terminated.
    ) else (
        echo [FAILED] Could not terminate process %HTTP_PID%.
    )
) else (
    echo Port %HTTP_PORT% is free.
)

echo.

:: 检查并终止IPC端口进程
echo Checking IPC port %IPC_PORT%...
set IPC_PID=
for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%IPC_PORT% ^| findstr LISTENING') do (
    set IPC_PID=%%a
)

if defined IPC_PID (
    echo Port %IPC_PORT% is occupied by process %IPC_PID%
    
    :: 获取进程信息
    for /f "tokens=2 delims=," %%a in ('wmic process where ProcessId^=%IPC_PID% get Name /format:csv 2^>nul ^| findstr /v "^$" ^| findstr /v "Node,Name"') do (
        echo Process name: %%a
    )
    
    echo Terminating process %IPC_PID%...
    taskkill /F /PID %IPC_PID% >nul 2>&1
    if !errorlevel! equ 0 (
        echo [SUCCESS] Process %IPC_PID% terminated.
    ) else (
        echo [FAILED] Could not terminate process %IPC_PID%.
    )
) else (
    echo Port %IPC_PID% is free.
)

echo.

:: 检查其他常用端口
echo Checking other commonly used ports...
set COMMON_PORTS=18080 19999 28080 29999

for %%p in (%COMMON_PORTS%) do (
    set FOUND=
    for /f "tokens=5" %%a in ('netstat -ano ^| findstr :%%p ^| findstr LISTENING') do (
        set FOUND=1
        echo Port %%p is occupied by process %%a
    )
    if not defined FOUND (
        echo Port %%p is free.
    )
)

echo.
echo ========================================
echo Port cleanup completed.
echo ========================================
echo.
pause

endlocal