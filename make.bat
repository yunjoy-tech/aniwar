@echo off

::go env -w GO111MODULE=on
::go env -w GOPROXY=https://goproxy.cn,https://gitlab.musadisca-games.com/wangxw,direct
::go env -w GOPRIVATE="*.musadisca-games.com"
::go env -w GONOSUMDB=gitlab.musadisca-games.com

set hour=
if %time:~0,2% leq 9 (set hour=0%time:~1,1%) else (set hour=%time:~0,2%)
set LDFLAGS="-ldflags=-X gitlab.musadisca-games.com/wangxw/musae/framework/global.APP_VERSION=DEBUG|"%date:~0,4%-%date:~5,2%-%date:~8,2%-%hour%-%time:~3,2%-%time:~6,2%

go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/guideserver.exe ./src/guideserver
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/loginserver.exe ./src/loginserver
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/gateserver.exe ./src/gateserver
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/lobbyserver.exe ./src/lobbyserver
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/actorserver.exe ./src/actorserver
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/billserver.exe ./src/billserver
go build -gcflags "-N -l" %LDFLAGS% -o ./output/bin/win/idipserver.exe ./src/idipserver

echo "ALL SERVERS build success!"