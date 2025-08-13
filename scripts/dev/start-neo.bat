@echo off
echo Starting Neo Core Framework...
cd ..\..\bin
start "Neo Core" neo.exe -ipc :9999
echo Neo Core started on IPC port 9999
timeout /t 2 >nul