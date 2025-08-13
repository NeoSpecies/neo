@echo off
echo ========================================
echo Selective Sync to Main Branch
echo ========================================
echo.

REM 请修改为你的主分支目录
set TARGET_DIR=C:\YourMainProject\neo-framework

echo Target directory: %TARGET_DIR%
echo.
echo Select what to sync:
echo [1] Core Framework Only (internal/, cmd-src/neo/)
echo [2] Gateway Services (services/http-gateway, services/tcp-gateway)
echo [3] Examples and Tests (examples-ipc/, test/)
echo [4] Documentation Only (docs/, *.md)
echo [5] Everything except binaries
echo [6] Custom selection
echo.

set /p choice="Enter your choice (1-6): "

if "%choice%"=="1" goto core_only
if "%choice%"=="2" goto gateways_only
if "%choice%"=="3" goto examples_only
if "%choice%"=="4" goto docs_only
if "%choice%"=="5" goto everything
if "%choice%"=="6" goto custom

:core_only
echo.
echo Syncing Core Framework...
xcopy /E /Y /I internal %TARGET_DIR%\internal
xcopy /E /Y /I cmd-src\neo %TARGET_DIR%\cmd-src\neo
xcopy /E /Y /I pkg %TARGET_DIR%\pkg
copy /Y go.mod %TARGET_DIR%\
copy /Y go.sum %TARGET_DIR%\
goto done

:gateways_only
echo.
echo Syncing Gateway Services...
xcopy /E /Y /I services\http-gateway %TARGET_DIR%\services\http-gateway
xcopy /E /Y /I services\tcp-gateway %TARGET_DIR%\services\tcp-gateway
xcopy /E /Y /I services\README.md %TARGET_DIR%\services\
goto done

:examples_only
echo.
echo Syncing Examples and Tests...
xcopy /E /Y /I examples-ipc %TARGET_DIR%\examples-ipc
xcopy /E /Y /I test %TARGET_DIR%\test
goto done

:docs_only
echo.
echo Syncing Documentation...
xcopy /E /Y /I docs %TARGET_DIR%\docs
copy /Y README.md %TARGET_DIR%\
copy /Y PROJECT_STRUCTURE.md %TARGET_DIR%\
goto done

:everything
echo.
echo Syncing Everything (except binaries)...
xcopy /E /Y /I cmd-src %TARGET_DIR%\cmd-src
xcopy /E /Y /I internal %TARGET_DIR%\internal
xcopy /E /Y /I services %TARGET_DIR%\services
xcopy /E /Y /I pkg %TARGET_DIR%\pkg
xcopy /E /Y /I examples-ipc %TARGET_DIR%\examples-ipc
xcopy /E /Y /I configs %TARGET_DIR%\configs
xcopy /E /Y /I scripts %TARGET_DIR%\scripts
xcopy /E /Y /I docs %TARGET_DIR%\docs
xcopy /E /Y /I test %TARGET_DIR%\test
copy /Y *.md %TARGET_DIR%\
copy /Y go.* %TARGET_DIR%\
goto done

:custom
echo.
echo Custom sync - select components:
set /p sync_core="Sync core framework? (y/n): "
set /p sync_gateways="Sync gateway services? (y/n): "
set /p sync_examples="Sync examples? (y/n): "
set /p sync_docs="Sync documentation? (y/n): "
set /p sync_scripts="Sync scripts? (y/n): "

if /i "%sync_core%"=="y" (
    xcopy /E /Y /I internal %TARGET_DIR%\internal
    xcopy /E /Y /I cmd-src\neo %TARGET_DIR%\cmd-src\neo
)
if /i "%sync_gateways%"=="y" (
    xcopy /E /Y /I services %TARGET_DIR%\services
)
if /i "%sync_examples%"=="y" (
    xcopy /E /Y /I examples-ipc %TARGET_DIR%\examples-ipc
)
if /i "%sync_docs%"=="y" (
    xcopy /E /Y /I docs %TARGET_DIR%\docs
    copy /Y *.md %TARGET_DIR%\
)
if /i "%sync_scripts%"=="y" (
    xcopy /E /Y /I scripts %TARGET_DIR%\scripts
)

:done
echo.
echo ========================================
echo Sync completed!
echo ========================================
echo.
echo Next steps:
echo 1. cd %TARGET_DIR%
echo 2. git status (review changes)
echo 3. git add -A
echo 4. git commit -m "Your commit message"
echo 5. git push origin main
echo ========================================