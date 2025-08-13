@echo off
echo ========================================
echo Sync Neo Framework to Main Project
echo Target: C:\Dev\neo
echo ========================================
echo.

set TARGET_DIR=C:\Dev\neo
set SOURCE_DIR=C:\Dev\light

echo Source: %SOURCE_DIR%
echo Target: %TARGET_DIR%
echo.

echo This will sync all changes to the main project.
echo Press Ctrl+C to cancel or
pause

echo.
echo [1/10] Syncing core framework source...
xcopy /E /Y /I /Q "%SOURCE_DIR%\internal" "%TARGET_DIR%\internal"

echo [2/10] Syncing command source (renamed from cmd to cmd-src)...
if exist "%TARGET_DIR%\cmd" (
    echo Backing up old cmd directory to cmd.backup...
    move "%TARGET_DIR%\cmd" "%TARGET_DIR%\cmd.backup" >nul 2>&1
)
xcopy /E /Y /I /Q "%SOURCE_DIR%\cmd-src" "%TARGET_DIR%\cmd-src"

echo [3/10] Syncing services (new gateway services)...
xcopy /E /Y /I /Q "%SOURCE_DIR%\services" "%TARGET_DIR%\services"

echo [4/10] Syncing package files...
xcopy /E /Y /I /Q "%SOURCE_DIR%\pkg" "%TARGET_DIR%\pkg"

echo [5/10] Syncing examples...
xcopy /E /Y /I /Q "%SOURCE_DIR%\examples-ipc" "%TARGET_DIR%\examples-ipc"

echo [6/10] Syncing configs...
xcopy /E /Y /I /Q "%SOURCE_DIR%\configs" "%TARGET_DIR%\configs"

echo [7/10] Syncing scripts...
xcopy /E /Y /I /Q "%SOURCE_DIR%\scripts" "%TARGET_DIR%\scripts"

echo [8/10] Syncing documentation...
xcopy /E /Y /I /Q "%SOURCE_DIR%\docs" "%TARGET_DIR%\docs"

echo [9/10] Syncing test files...
xcopy /E /Y /I /Q "%SOURCE_DIR%\test" "%TARGET_DIR%\test"

echo [10/10] Syncing root files...
copy /Y "%SOURCE_DIR%\README.md" "%TARGET_DIR%\" >nul
copy /Y "%SOURCE_DIR%\PROJECT_STRUCTURE.md" "%TARGET_DIR%\" >nul
copy /Y "%SOURCE_DIR%\go.mod" "%TARGET_DIR%\" >nul
copy /Y "%SOURCE_DIR%\go.sum" "%TARGET_DIR%\" >nul
copy /Y "%SOURCE_DIR%\go.work" "%TARGET_DIR%\" >nul
copy /Y "%SOURCE_DIR%\.gitignore" "%TARGET_DIR%\" >nul

echo.
echo Creating bin directory if not exists...
if not exist "%TARGET_DIR%\bin" mkdir "%TARGET_DIR%\bin"

echo.
echo ========================================
echo Sync completed successfully!
echo ========================================
echo.
echo Files synced:
echo - Core framework (internal/)
echo - Command sources (cmd-src/)
echo - Gateway services (services/)
echo - Examples (examples-ipc/)
echo - Scripts (scripts/)
echo - Documentation (docs/, *.md)
echo - Go module files
echo.
echo NOT synced (intentionally):
echo - Binary files (*.exe)
echo - Log files
echo - Temporary files
echo.
echo Next: Run build and test in main project
echo ========================================