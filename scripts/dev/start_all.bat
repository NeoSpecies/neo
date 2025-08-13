@echo off
echo ====================================
echo  Neo Framework Services Launcher
echo ====================================
echo.

REM 检查Neo核心是否在运行
echo Checking if Neo core is running...
netstat -an | findstr :9999 >nul
if %errorlevel% neq 0 (
    echo [ERROR] Neo core is not running on port 9999
    echo Please start Neo core first with: neo.exe -ipc :9999
    echo.
    pause
    exit /b 1
)
echo [OK] Neo core is running

REM 编译所有服务
echo.
echo Building services...

echo Building HTTP Gateway...
cd http-gateway
go build -o http-gateway.exe main.go service.go
if %errorlevel% neq 0 (
    echo [ERROR] Failed to build HTTP Gateway
    pause
    exit /b 1
)
echo [OK] HTTP Gateway built

echo Building TCP Gateway...
cd ..\tcp-gateway
go build -o tcp-gateway.exe main.go service.go
if %errorlevel% neq 0 (
    echo [ERROR] Failed to build TCP Gateway
    pause
    exit /b 1
)
echo [OK] TCP Gateway built
cd ..

REM 启动服务
echo.
echo Starting services...

echo Starting HTTP Gateway on port 8080...
start "HTTP Gateway" cmd /c "cd http-gateway && http-gateway.exe -ipc localhost:9999 -http :8080"

echo Starting TCP Gateway on port 7777...
start "TCP Gateway" cmd /c "cd tcp-gateway && tcp-gateway.exe -ipc localhost:9999 -tcp :7777"

REM 等待服务启动
timeout /t 3 /nobreak >nul

echo.
echo ====================================
echo  Services Started Successfully!
echo ====================================
echo.
echo Services running:
echo   - HTTP Gateway: http://localhost:8080
echo     API: http://localhost:8080/api/{service}/{method}
echo     Health: http://localhost:8080/health
echo.
echo   - TCP Gateway: localhost:7777
echo     Protocol: JSON with length prefix
echo.
echo Press any key to stop all services...
pause >nul

REM 停止服务
echo.
echo Stopping services...
taskkill /FI "WINDOWTITLE eq HTTP Gateway" /F >nul 2>&1
taskkill /FI "WINDOWTITLE eq TCP Gateway" /F >nul 2>&1

echo Services stopped.
pause