@echo off
echo ========================================
echo Sync Neo Framework to Main Branch
echo ========================================
echo.

REM 设置目标目录（请修改为你的主分支目录）
set TARGET_DIR=C:\YourMainProject\neo-framework
echo Target directory: %TARGET_DIR%
echo.

echo WARNING: This will copy files to your main branch directory.
echo Please make sure the target directory is correct!
pause

echo.
echo Copying source code...
REM 复制核心源代码
xcopy /E /Y /I cmd-src %TARGET_DIR%\cmd-src
xcopy /E /Y /I internal %TARGET_DIR%\internal
xcopy /E /Y /I services %TARGET_DIR%\services
xcopy /E /Y /I pkg %TARGET_DIR%\pkg

echo Copying examples...
xcopy /E /Y /I examples-ipc %TARGET_DIR%\examples-ipc

echo Copying configuration...
xcopy /E /Y /I configs %TARGET_DIR%\configs

echo Copying scripts...
xcopy /E /Y /I scripts %TARGET_DIR%\scripts

echo Copying documentation...
xcopy /E /Y /I docs %TARGET_DIR%\docs
copy /Y README.md %TARGET_DIR%\
copy /Y PROJECT_STRUCTURE.md %TARGET_DIR%\

echo Copying Go module files...
copy /Y go.mod %TARGET_DIR%\
copy /Y go.sum %TARGET_DIR%\
copy /Y go.work %TARGET_DIR%\

echo Copying test files...
xcopy /E /Y /I test %TARGET_DIR%\test

echo.
echo ========================================
echo Files copied successfully!
echo ========================================
echo.
echo IMPORTANT: Files NOT copied (intentionally):
echo - bin/ directory (compiled binaries)
echo - logs/ directory
echo - *.exe files
echo - .git/ directory
echo - temporary files
echo.
echo Next steps:
echo 1. cd %TARGET_DIR%
echo 2. Review the changes: git status
echo 3. Add files: git add .
echo 4. Commit: git commit -m "feat: Update Neo Framework to v3.0 architecture"
echo 5. Push to remote: git push origin main
echo ========================================