@echo off
echo Testing Neo Framework IPC Examples
echo ==================================

REM Test Python
echo.
echo [1/5] Testing Python client...
cd examples-ipc\python
start /B python service.py
timeout /t 3 >nul
curl -X POST http://localhost:8080/api/demo-service/hello -H "Content-Type: application/json" -d "{\"name\": \"Python\"}" 2>nul
if %errorlevel% neq 0 (
    echo Python test FAILED
) else (
    echo Python test PASSED
)
taskkill /F /IM python.exe >nul 2>&1
cd ..\..

REM Test Go  
echo.
echo [2/5] Testing Go client...
cd examples-ipc\go
start /B go run service.go
timeout /t 3 >nul
curl -X POST http://localhost:8080/api/demo-service/hello -H "Content-Type: application/json" -d "{\"name\": \"Go\"}" 2>nul
if %errorlevel% neq 0 (
    echo Go test FAILED
) else (
    echo Go test PASSED
)
taskkill /F /IM go.exe >nul 2>&1
cd ..\..

REM Test Node.js
echo.
echo [3/5] Testing Node.js client...
cd examples-ipc\nodejs
start /B node service.js
timeout /t 3 >nul
curl -X POST http://localhost:8080/api/demo-service/hello -H "Content-Type: application/json" -d "{\"name\": \"Node.js\"}" 2>nul
if %errorlevel% neq 0 (
    echo Node.js test FAILED
) else (
    echo Node.js test PASSED
)
taskkill /F /IM node.exe >nul 2>&1
cd ..\..

REM Test Java (需要先编译)
echo.
echo [4/5] Testing Java client...
cd examples-ipc\java
echo Compiling Java...
javac Service.java 2>nul
if %errorlevel% neq 0 (
    echo Java compilation FAILED - Gson library may be missing
) else (
    start /B java Service
    timeout /t 3 >nul
    curl -X POST http://localhost:8080/api/demo-service/hello -H "Content-Type: application/json" -d "{\"name\": \"Java\"}" 2>nul
    if %errorlevel% neq 0 (
        echo Java test FAILED
    ) else (
        echo Java test PASSED
    )
    taskkill /F /IM java.exe >nul 2>&1
)
cd ..\..

REM Test PHP
echo.
echo [5/5] Testing PHP client...
cd examples-ipc\php
start /B php service.php
timeout /t 3 >nul
curl -X POST http://localhost:8080/api/demo-service/hello -H "Content-Type: application/json" -d "{\"name\": \"PHP\"}" 2>nul
if %errorlevel% neq 0 (
    echo PHP test FAILED
) else (
    echo PHP test PASSED
)
taskkill /F /IM php.exe >nul 2>&1
cd ..\..

echo.
echo ==================================
echo All tests completed!