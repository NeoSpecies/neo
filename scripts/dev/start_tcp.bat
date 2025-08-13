@echo off
echo ====================================
echo  TCP Gateway Service
echo ====================================
echo.

REM 检查参数
set IPC_ADDR=localhost:9999
set TCP_ADDR=:7777
set PROTOCOL=json

if "%1" neq "" set IPC_ADDR=%1
if "%2" neq "" set TCP_ADDR=%2
if "%3" neq "" set PROTOCOL=%3

REM 编译服务
echo Building TCP Gateway...
cd tcp-gateway
go build -o tcp-gateway.exe main.go service.go
if %errorlevel% neq 0 (
    echo [ERROR] Failed to build TCP Gateway
    pause
    exit /b 1
)

echo.
echo Starting TCP Gateway...
echo   IPC Server: %IPC_ADDR%
echo   TCP Port:   %TCP_ADDR%
echo   Protocol:   %PROTOCOL%
echo.

tcp-gateway.exe -ipc %IPC_ADDR% -tcp %TCP_ADDR% -protocol %PROTOCOL%