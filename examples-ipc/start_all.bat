@echo off
echo Starting all IPC services...

echo Starting Neo Framework...
start "Neo Framework" cmd /c "cd /d C:\Dev\neo && neo.exe"
timeout /t 5 > nul

echo Starting Java service...
start "Java Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\java && java -cp .;gson-2.10.1.jar Service"
timeout /t 3 > nul

echo Starting Python service...
start "Python Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\python && python service.py"
timeout /t 3 > nul

echo Starting Go service...
start "Go Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\go && go run service.go"
timeout /t 3 > nul

echo Starting PHP service...
start "PHP Service" cmd /c "cd /d C:\Dev\neo\examples-ipc\php && php service.php"
timeout /t 3 > nul

echo All services started!