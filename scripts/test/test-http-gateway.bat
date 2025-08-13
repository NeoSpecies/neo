@echo off
echo ========================================
echo Testing HTTP Gateway Service
echo ========================================
echo.

echo [Test 1] Calculate - Addition
curl -X POST http://localhost:8081/api/demo-service-python/calculate -H "Content-Type: application/json" -d "{\"operation\":\"add\",\"a\":10,\"b\":20}"
echo.
echo.

echo [Test 2] Calculate - Multiplication
curl -X POST http://localhost:8081/api/demo-service-python/calculate -H "Content-Type: application/json" -d "{\"operation\":\"multiply\",\"a\":7,\"b\":8}"
echo.
echo.

echo [Test 3] Echo
curl -X POST http://localhost:8081/api/demo-service-python/echo -H "Content-Type: application/json" -d "{\"message\":\"Hello from test script!\"}"
echo.
echo.

echo [Test 4] HTTP Gateway Info
curl -X POST http://localhost:8081/api/http-gateway/getInfo -H "Content-Type: application/json" -d "{}"
echo.
echo.

echo [Test 5] TCP Gateway Info (via HTTP)
curl -X POST http://localhost:8081/api/tcp-gateway/getInfo -H "Content-Type: application/json" -d "{}"
echo.
echo.

echo ========================================
echo HTTP Gateway tests completed!
echo ========================================