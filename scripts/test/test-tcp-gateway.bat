@echo off
echo ========================================
echo Testing TCP Gateway Service
echo ========================================
echo.

cd ..\..\bin

if exist test_client.exe (
    echo Running TCP test client...
    test_client.exe localhost:7777
) else (
    echo Error: test_client.exe not found in bin directory
    echo Please run build-all.bat first
)

echo.
echo ========================================
echo TCP Gateway tests completed!
echo ========================================