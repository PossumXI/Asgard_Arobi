@echo off
REM ASGARD Demo Recorder - Quick Start
REM Run this batch file to record demos of all ASGARD systems

echo.
echo ========================================
echo    ASGARD Demo Recorder
echo ========================================
echo.

cd /d "%~dp0"

REM Check if node_modules exists
if not exist "node_modules" (
    echo Installing dependencies...
    call npm install
    if errorlevel 1 (
        echo ERROR: Failed to install dependencies
        pause
        exit /b 1
    )
    echo Installing Playwright browsers...
    call npx playwright install chromium
    if errorlevel 1 (
        echo ERROR: Failed to install Playwright
        pause
        exit /b 1
    )
)

echo Starting demo recording...
echo.

call npx ts-node demo-recorder.ts %*

echo.
echo Recording complete! Check demo-output folder.
pause
