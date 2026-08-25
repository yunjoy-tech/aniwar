@echo off
rem ============================================================
rem Consul and Dapr Placement Container Startup Script
rem Make sure Docker Desktop is running before executing
rem ============================================================

echo [INFO] Starting Consul and Dapr Placement containers...

:: Check if Docker is running
docker info >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Docker is not running. Please start Docker Desktop first.
    pause
    exit /b 1
)

:: Stop existing containers if any
docker stop consul dapr-placement 2>nul
docker rm consul dapr-placement 2>nul

:: Start Consul (use specific version)
echo [INFO] Starting Consul on port 8500...
docker run -d --name consul -p 8500:8500 consul:1.15.3 agent -dev -ui -client 0.0.0.0

if %ERRORLEVEL% neq 0 (
    echo [ERROR] Failed to start Consul.
    pause
    exit /b 1
)

:: Start Dapr Placement (use specific version with command)
echo [INFO] Starting Dapr Placement on port 6050...
docker run -d --name dapr-placement -p 6050:6050 -p 9091:9091 daprio/placement:1.10.4 ./placement

if %ERRORLEVEL% neq 0 (
    echo [ERROR] Failed to start Dapr Placement.
    docker stop consul
    docker rm consul
    pause
    exit /b 1
)

echo.
echo ============================================================
echo [SUCCESS] Services started successfully!
echo ============================================================
echo.
echo   Consul UI:      http://localhost:8500
echo   Placement:      localhost:6050
echo.
echo   Press any key to exit. Services will keep running in background.
echo ============================================================

pause >nul