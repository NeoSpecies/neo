@echo off
echo Starting Gateway Services...
echo.

cd ..\..\bin

echo Starting HTTP Gateway on port 8081...
start "HTTP Gateway" http-gateway.exe -ipc localhost:9999 -http :8081
timeout /t 1 >nul

echo Starting TCP Gateway on port 7777...
start "TCP Gateway" tcp-gateway.exe -ipc localhost:9999 -tcp :7777
timeout /t 1 >nul

echo.
echo Gateway services started:
echo - HTTP Gateway: http://localhost:8081
echo - TCP Gateway:  localhost:7777
echo.