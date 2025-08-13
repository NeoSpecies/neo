@echo off
echo ========================================
echo Create Git Patches for Main Branch
echo ========================================
echo.

REM 创建patches目录
if not exist patches mkdir patches

echo Creating patches for recent commits...
echo.

REM 获取最近的提交并创建补丁
echo [1] Create patches for last 5 commits
git format-patch -5 -o patches/

echo.
echo Patches created in patches/ directory:
dir patches\*.patch /B

echo.
echo ========================================
echo How to apply patches in main branch:
echo ========================================
echo.
echo 1. Copy patches/ directory to your main project
echo 2. In main project directory, run:
echo    git am patches/*.patch
echo.
echo Or apply selectively:
echo    git am patches/0001-specific-commit.patch
echo.
echo To check before applying:
echo    git apply --check patches/*.patch
echo.
echo To see what will change:
echo    git apply --stat patches/*.patch
echo ========================================