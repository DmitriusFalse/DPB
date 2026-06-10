@echo off
setlocal enabledelayedexpansion

echo === Danbooru Prompt Builder ===
echo.

:: Check if Go is installed
go version >nul 2>nul
if errorlevel 1 (
    echo Go not found.
    echo Download from https://go.dev/dl/ and install Go, then run this script again.
    pause
    exit /b 1
)

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

set BUILD_DIR=build
if not exist "%BUILD_DIR%" mkdir "%BUILD_DIR%"

echo [1/4] Downloading dependencies...
call go mod download
if %errorlevel% neq 0 (
    echo Error downloading dependencies
    pause
    exit /b 1
)

echo [2/4] Installing rsrc for icon embedding...
go install github.com/akavel/rsrc@latest
if %errorlevel% neq 0 (
    echo Error installing rsrc
    pause
    exit /b 1
)

echo [3/4] Generating icon resources...
rsrc -ico tray\icon.ico -o rsrc.syso
if %errorlevel% neq 0 (
    echo Error generating resources
    pause
    exit /b 1
)

:: Increment build number
set VERSION_FILE=version.txt
if exist "%VERSION_FILE%" (
    set /p VERSION=<"%VERSION_FILE%"
    for /f "tokens=1,2,3 delims=." %%a in ("!VERSION!") do (
        set MAJOR=%%a
        set MINOR=%%b
        set BUILD=%%c
    )
    set /a BUILD+=1
    >"%VERSION_FILE%" echo !MAJOR!.!MINOR!.!BUILD!
)

echo [4/4] Building binary...
:: -s -w: strip debug info
:: -H=windowsgui: hide console window
:: -trimpath: remove build paths
go build -ldflags="-s -w -H=windowsgui" -trimpath -o "%BUILD_DIR%\DanbooruPromptBuilder.exe"
if %errorlevel% neq 0 (
    echo Build failed
    del rsrc.syso 2>nul
    pause
    exit /b 1
)

del rsrc.syso

:: Copy config alongside the binary
copy /Y config.json "%BUILD_DIR%\config.json" >nul

:: Copy tags folder alongside the binary
if exist "tags" (
    if exist "%BUILD_DIR%\tags" rd /s /q "%BUILD_DIR%\tags"
    xcopy /E /I /Q "tags" "%BUILD_DIR%\tags" >nul
)

:: Copy static images alongside the binary
if exist "handler\img" (
    if exist "%BUILD_DIR%\img" rd /s /q "%BUILD_DIR%\img"
    xcopy /E /I /Q "handler\img" "%BUILD_DIR%\img" >nul
)

:: Copy workflows alongside the binary
if exist "Workflows" (
    if exist "%BUILD_DIR%\Workflows" rd /s /q "%BUILD_DIR%\Workflows"
    xcopy /E /I /Q "Workflows" "%BUILD_DIR%\Workflows" >nul
)

:: Clean up junk files left after tests
if exist "%BUILD_DIR%\*.log" del "%BUILD_DIR%\*.log"

echo.
echo Done! Binary: %BUILD_DIR%\DanbooruPromptBuilder.exe

for %%f in ("%BUILD_DIR%\DanbooruPromptBuilder.exe") do (
    echo Size: %%~zf bytes
)

endlocal
pause
