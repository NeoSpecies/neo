@echo off
REM Neo Framework 测试运行脚本 (Windows)

echo Neo Framework 测试套件
echo =========================

if "%1"=="python" (
    echo 运行Python测试...
    python test\python\test_simple_client.py
    python test\python\test_ipc_client.py
) else if "%1"=="stress" (
    echo 运行压力测试...
    python test\stress\test_stress.py
) else if "%1"=="integration" (
    echo 运行Go集成测试...
    go run test\integration\test_list_services.go
) else if "%1"=="all" (
    echo 运行所有测试...
    echo.
    echo [1/3] Python测试
    python test\python\test_simple_client.py
    echo.
    echo [2/3] Go集成测试
    go run test\integration\test_list_services.go
    echo.
    echo [3/3] 压力测试
    python test\stress\test_stress.py
) else (
    echo 使用方法: test\run_tests.bat [python^|stress^|integration^|all]
    echo   python      - 运行Python客户端测试
    echo   stress      - 运行压力测试
    echo   integration - 运行Go集成测试
    echo   all         - 运行所有测试
)