#!/bin/bash
# Neo Framework Quick Start
# This is a shortcut to the main startup script

echo "Starting Neo Framework..."
echo "========================="
echo ""
echo "Starting main application on ports:"
echo "  HTTP Gateway: 28080"
echo "  IPC Server: 29999"
echo ""
go run cmd/neo/main.go