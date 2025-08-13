@echo off
echo ====================================
echo  HTTP Gateway Service
echo ====================================
echo.

REM 检查参数
set IPC_ADDR=localhost:9999
set HTTP_ADDR=:8080

if "%1" neq "" set IPC_ADDR=%1
if "%2" neq "" set HTTP_ADDR=%2

REM 编译服务
echo Building HTTP Gateway...
cd http-gateway
go build -o http-gateway.exe main.go service.go
if %errorlevel% neq 0 (
    echo [ERROR] Failed to build HTTP Gateway
    pause
    exit /b 1
)

echo.
echo Starting HTTP Gateway...
echo   IPC Server: %IPC_ADDR%
echo   HTTP Port:  %HTTP_ADDR%
echo.

http-gateway.exe -ipc %IPC_ADDR% -http %HTTP_ADDR%