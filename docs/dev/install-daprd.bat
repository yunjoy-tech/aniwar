@echo off
set TOOLSET_DIR=../../../toolset

:: !!!需要先安装Dapr Cli程序!!!

:: 卸载 Dapr 并等待完成
start /wait dapr.exe uninstall --all
echo Dapr uninstalled successfully

:: 等待 3 秒（让端口/进程彻底释放）
timeout /t 3 /nobreak >nul

:: 初始化 Dapr（从本地 bundle）
start /wait dapr.exe init -s --from-dir %TOOLSET_DIR%\daprbundle\win
echo Dapr initialized successfully

:: 查看版本（验证是否成功）
dapr.exe version

:: 再等 3 秒（让 placement / Redis 启动就绪）
timeout /t 3 /nobreak >nul