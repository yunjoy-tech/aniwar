@echo off
rem ============================================================
rem Consul 和 Dapr Placement 容器停止脚本
rem ============================================================

echo [INFO] Stopping Consul and Dapr Placement containers...

docker stop consul dapr-placement 2>nul
docker rm consul dapr-placement 2>nul

echo [SUCCESS] Containers stopped and removed.