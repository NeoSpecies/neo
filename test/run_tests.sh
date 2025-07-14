#!/bin/bash
# Neo Framework 测试运行脚本

echo "🧪 Neo Framework 测试套件"
echo "========================="

# 检查参数
if [ "$1" == "python" ]; then
    echo "运行Python测试..."
    python test/python/test_simple_client.py
    python test/python/test_ipc_client.py
elif [ "$1" == "stress" ]; then
    echo "运行压力测试..."
    python test/stress/test_stress.py
elif [ "$1" == "integration" ]; then
    echo "运行Go集成测试..."
    go run test/integration/test_list_services.go
elif [ "$1" == "all" ]; then
    echo "运行所有测试..."
    echo -e "\n[1/3] Python测试"
    python test/python/test_simple_client.py
    echo -e "\n[2/3] Go集成测试"
    go run test/integration/test_list_services.go
    echo -e "\n[3/3] 压力测试"
    python test/stress/test_stress.py
else
    echo "使用方法: ./test/run_tests.sh [python|stress|integration|all]"
    echo "  python      - 运行Python客户端测试"
    echo "  stress      - 运行压力测试"
    echo "  integration - 运行Go集成测试"
    echo "  all         - 运行所有测试"
fi