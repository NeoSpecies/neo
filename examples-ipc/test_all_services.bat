@echo off
echo === Starting All IPC Services ===
echo.

REM Start Go service
echo Starting Go service...
start "Go Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\go && go run service.go"
timeout /t 3 > nul

REM Start Java service
echo Starting Java service...
start "Java Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\java && java -cp .;gson-2.10.1.jar Service"
timeout /t 3 > nul

REM Start Python service
echo Starting Python service...
start "Python Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\python && python service.py"
timeout /t 3 > nul

REM Start PHP service
echo Starting PHP service...
start "PHP Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\php && php service.php"
timeout /t 3 > nul

echo.
echo === Testing All Services ===
echo.

echo [Go Service Test]
curl -s -X POST "http://localhost:8080/api/demo-service-go/hello" -H "Content-Type: application/json" -d "{\"name\": \"Go\"}"
echo.
echo.

echo [Java Service Test]
curl -s -X POST "http://localhost:8080/api/demo-service-java/hello" -H "Content-Type: application/json" -d "{\"name\": \"Java\"}"
echo.
echo.

echo [Python Service Test]
curl -s -X POST "http://localhost:8080/api/demo-service-python/hello" -H "Content-Type: application/json" -d "{\"name\": \"Python\"}"
echo.
echo.

echo [PHP Service Test]
curl -s -X POST "http://localhost:8080/api/demo-service-php/hello" -H "Content-Type: application/json" -d "{\"name\": \"PHP\"}"
echo.
echo.

echo === Test Complete ===
pause