@echo off

::go env -w GO111MODULE=on
::go env -w GOPROXY=https://goproxy.cn,https://gitlab.musadisca-games.com/wangxw,direct
::go env -w GOPRIVATE="*.musadisca-games.com"
::go env -w GONOSUMDB=gitlab.musadisca-games.com

::set hour=
::if %time:~0,2% leq 9 (set hour=0%time:~1,1%) else (set hour=%time:~0,2%)
::set LDFLAGS="-ldflags=-X gitee.com/aniwar2/musae/framework/global.APP_VERSION=DEBUG|"%date:~0,4%-%date:~5,2%-%date:~8,2%-%hour%-%time:~3,2%-%time:~6,2%
set LDFLAGS="-ldflags=-X gitee.com/aniwar2/musae/framework/global.APP_VERSION=DEBUG"

echo "Build ActorServer"
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/actorserver.exe ./src/actorserver || exit /b 1
echo "Build GuideServer"
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/guideserver.exe ./src/guideserver || exit /b 1
echo "Build LoginServer"
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/loginserver.exe ./src/loginserver || exit /b 1
echo "Build GateServer"
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/gateserver.exe ./src/gateserver || exit /b 1
echo "Build LobbyServer"
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/lobbyserver.exe ./src/lobbyserver || exit /b 1
echo "Build BillServer"
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/billserver.exe ./src/billserver || exit /b 1
echo "Build IdipServer"
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/idipserver.exe ./src/idipserver || exit /b 1

echo "ALL SERVERS build success!"