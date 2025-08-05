@echo off
echo === Neo IPC Services Final Test ===
echo.

echo [1/4] Testing Java Service...
curl -s -X POST "http://localhost:8080/api/demo-service-java/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" > nul 2>&1
if %errorlevel% == 0 (
    echo Java: PASS
    curl -X POST "http://localhost:8080/api/demo-service-java/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" 2>nul
) else (
    echo Java: FAIL
)
echo.

echo [2/4] Testing Python Service...
curl -s -X POST "http://localhost:8080/api/demo-service-python/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" > nul 2>&1
if %errorlevel% == 0 (
    echo Python: PASS
    curl -X POST "http://localhost:8080/api/demo-service-python/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" 2>nul
) else (
    echo Python: FAIL
)
echo.

echo [3/4] Testing Go Service...
curl -s -X POST "http://localhost:8080/api/demo-service-go/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" > nul 2>&1
if %errorlevel% == 0 (
    echo Go: PASS
    curl -X POST "http://localhost:8080/api/demo-service-go/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" 2>nul
) else (
    echo Go: FAIL
)
echo.

echo [4/4] Testing PHP Service...
curl -s -X POST "http://localhost:8080/api/demo-service-php/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" > nul 2>&1
if %errorlevel% == 0 (
    echo PHP: PASS
    curl -X POST "http://localhost:8080/api/demo-service-php/hello" -H "Content-Type: application/json" -d "{\"name\": \"Test\"}" 2>nul
) else (
    echo PHP: FAIL
)
echo.

echo === Test Complete ===