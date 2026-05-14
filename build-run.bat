@echo off
echo ========================================
echo   logs-server Build and Run Script
echo ========================================
echo.

echo [1/3] Stopping logs-server.exe...
taskkill /F /IM logs-server.exe >nul 2>&1
if %errorlevel% equ 0 (
    echo   - Process stopped
) else (
    echo   - Process not running or already stopped
)
echo.

echo [2/3] Building project...
go build -o logs-server.exe ./cmd/server/
if %errorlevel% neq 0 (
    echo   - Build FAILED!
    pause
    exit /b 1
)
echo   - Build SUCCESS
echo.

echo [3/3] Starting logs-server.exe...
start "" logs-server.exe
echo   - Service started
echo.

echo ========================================
echo   Done!
echo ========================================
