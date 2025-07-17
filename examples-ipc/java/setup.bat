@echo off
echo ========================================
echo Neo Framework Java Service Setup
echo ========================================
echo.

:: Check if Java is installed
java -version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Java is not installed or not in PATH
    echo Please install Java 8 or higher
    pause
    exit /b 1
)

echo [INFO] Java is installed
echo.

:: Check if Gson exists
if not exist "gson-2.10.1.jar" (
    echo [INFO] Gson library not found. Downloading...
    echo.
    
    :: Try PowerShell download
    powershell -Command "(New-Object Net.WebClient).DownloadFile('https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar', 'gson-2.10.1.jar')" 2>nul
    
    if exist "gson-2.10.1.jar" (
        echo [SUCCESS] Gson library downloaded successfully
    ) else (
        echo [ERROR] Failed to download Gson library
        echo Please download manually from:
        echo https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar
        pause
        exit /b 1
    )
) else (
    echo [INFO] Gson library found
)

echo.
echo [INFO] Compiling Service.java...
javac -cp gson-2.10.1.jar Service.java

if %errorlevel% equ 0 (
    echo [SUCCESS] Compilation successful
    echo.
    echo To run the service, use:
    echo java -cp ".;gson-2.10.1.jar" Service
) else (
    echo [ERROR] Compilation failed
    pause
    exit /b 1
)

echo.
pause