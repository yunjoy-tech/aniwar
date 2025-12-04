@echo off
:: install-g.bat —— 一键安装 g.cmd 并初始化 PATH
setlocal

set "BIN_DIR=%USERPROFILE%\bin"
set "G_CMD_URL=https://raw.githubusercontent.com/voidint/g/master/g.cmd"

if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"

echo [1/3] 下载 g.cmd 到 %BIN_DIR%...
powershell -Command "Invoke-WebRequest -Uri '%G_CMD_URL%' -OutFile '%BIN_DIR%\g.cmd'"

if errorlevel 1 (
    echo [✗] 下载失败！请检查网络或手动下载： %G_CMD_URL%
    exit /b 1
)

echo [2/3] 检查 PATH 是否包含 %BIN_DIR%...
set "CURRENT_PATH=%PATH%"
echo %CURRENT_PATH% | find /i "%BIN_DIR%" >nul
if %errorlevel% equ 0 (
    echo     ✓ 已在 PATH 中
) else (
    echo [3/3] 正在添加 %BIN_DIR% 到用户 PATH...
    powershell -Command "[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';%BIN_DIR%', 'User')"
    echo     ✓ 已添加，请重启 CMD/PowerShell 生效
)

echo.
echo 🎉 安装完成！
echo 请关闭当前窗口，重新打开 CMD/PowerShell，然后运行：
echo      g ls-remote
echo      g install 1.22.4
echo      g ls
echo      g use 1.22.4
echo      go version

pause