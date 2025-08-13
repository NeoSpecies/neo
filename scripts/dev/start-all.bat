@echo off
echo ========================================
echo Starting Neo Framework Complete System
echo ========================================
echo.

REM Check if binaries exist
if not exist ..\bin\neo.exe (
    echo Error: neo.exe not found. Running build script...
    call build\build-all.bat
    if %ERRORLEVEL% NEQ 0 exit /b 1
)

echo [1/3] Starting Neo Core...
call dev\start-neo.bat

echo [2/3] Starting Example Services...
cd ..\examples-ipc\python
start "Python Demo Service" python service.py
cd ..\..\scripts
timeout /t 2 >nul

echo [3/3] Starting Gateway Services...
call dev\start-gateways.bat

echo.
echo ========================================
echo System Startup Complete!
echo ========================================
echo.
echo Available Services:
echo - Neo Core IPC:    localhost:9999
echo - HTTP Gateway:    http://localhost:8081
echo - TCP Gateway:     localhost:7777
echo - Python Service:  demo-service-python
echo.
echo Test Commands:
echo - scripts\test\test-http-gateway.bat
echo - scripts\test\test-tcp-gateway.bat
echo.
echo Stop all services by closing their windows
echo ========================================