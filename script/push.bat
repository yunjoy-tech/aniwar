@echo off
set TOOLSET_DIR=../../toolset
set SERVER_DIR=..\debug

rem 服务器私服程序svn推送

rem pull svn
if exist %SERVER_DIR% (
  echo "=======svn up======="
  %TOOLSET_DIR%\svn\svn.exe up %SERVER_DIR%
) else (
  echo "=======svn co======="
  %TOOLSET_DIR%\svn\svn.exe co svn://192.168.2.15/server/debug %SERVER_DIR%
)

rem commit svn
copy .\output\bin\win\actorserver.exe %SERVER_DIR%\output\bin\win\
copy .\output\bin\win\billserver.exe %SERVER_DIR%\output\bin\win\
copy .\output\bin\win\gateserver.exe %SERVER_DIR%\output\bin\win\
copy .\output\bin\win\guideserver.exe %SERVER_DIR%\output\bin\win\
copy .\output\bin\win\idipserver.exe %SERVER_DIR%\output\bin\win\
copy .\output\bin\win\lobbyserver.exe %SERVER_DIR%\output\bin\win\
copy .\output\bin\win\loginserver.exe %SERVER_DIR%\output\bin\win\

cd %SERVER_DIR%
%TOOLSET_DIR%\svn\svn.exe ci -m "push.bat auto commit"