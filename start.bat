@echo off
rem start.bat [d] 开启调试,启动actor的sidecar daprd

SETLOCAL ENABLEDELAYEDEXPANSION

set Debug="%1"

echo %Debug%
set GUIDE_APPID="guide"
set LOGIN_APPID="login"
rem random gate index
set GATE_APPID="gate_%username%"
set LOBBY_APPID="lobby"
set ACTOR_APPID="actor"
set BATTLE_APPID="battle"
set BILL_APPID="bill"
set IDIP_APPID="idip"
set ROBOT_APPID="robot"
set MUSAE_APPID="musae"

set PROTOCOL="http"

set GUIDE_APP_PORT=20001
set GUIDE_HTTP_PORT=20002
set GUIDE_GRPC_PORT=20003
set GUIDE_PPROF_PORT=20004

set LOGIN_OUT_PORT=12001
set LOGIN_APP_PORT=21001
set LOGIN_HTTP_PORT=21002
set LOGIN_GRPC_PORT=21003
set LOGIN_PPROF_PORT=21004

rem GATE_OUT_PORT和server.conf中gatePort参数保持一致
set GATE_OUT_PORT=13001
set GATE_APP_PORT=22001
set GATE_HTTP_PORT=22002
set GATE_GRPC_PORT=22003
set GATE_PPROF_PORT=22004

set LOBBY_APP_PORT=23001
set LOBBY_HTTP_PORT=23002
set LOBBY_GRPC_PORT=23003
set LOBBY_PPROF_PORT=23004

set ACTOR_APP_PORT=24001
set ACTOR_HTTP_PORT=24002
rem 为满足debug调试 grpc port必须是50001
set ACTOR_GRPC_PORT=24003
set ACTOR_PPROF_PORT=24004

set MAIL_APP_PORT=25001
set MAIL_HTTP_PORT=25002
set MAIL_GRPC_PORT=25003
set MAIL_PPROF_PORT=25004

set BATTLE_APP_PORT=27001
set BATTLE_HTTP_PORT=27002
set BATTLE_GRPC_PORT=27003
set BATTLE_PPROF_PORT=27004

set BILL_APP_PORT=28001
set BILL_HTTP_PORT=28002
set BILL_GRPC_PORT=28003
set BILL_PPROF_PORT=28004

set IDIP_APP_PORT=29001
set IDIP_HTTP_PORT=29002
set IDIP_GRPC_PORT=29003
set IDIP_PPROF_PORT=29004

set BIN_PATH=.\output\bin\win
set COMPONENT_PATH=.\output\cfg\component
set DAPR_CONFIG=.\output\cfg\dapr-config.yaml

set ACTORS="UserActor|RoomActor|AllianceActor|CenterActor|MailActor"

set RDSCFGHOST=aniwar-dev-global.redis.rds.aliyuncs.com:36379
set RDSCFGPASS=MFhqTGF3RG1UZm81SVZlUA==
set RDSCFGNS=cn
set RDSCFGGROUP=pob

wmic process where (name="consul.exe") get ProcessId | find /i "ProcessId" >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo "result:"%ERRORLEVEL%
    start "consul" /min %BIN_PATH%\consul.exe agent -dev -ui -client 0.0.0.0
    ping 127.0.0.1 -n 10
)

wmic process where (name="placement.exe") get ProcessId | find /i "ProcessId" >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo "result:"%ERRORLEVEL%
   start "placement" /min %BIN_PATH%\placement.exe -port 6050 -metrics-port 9091
    ping 127.0.0.1 -n 5
)

:: start "filebeat" /min ./output/bin/win/filebeat.exe -e -c ./output/cfg/filebeat.yml --path.data  ./filebeat --path.home ./ --path.logs ./log
:: start "filebeat-prom" /min ./output/bin/win/filebeat-prom.exe -e -c ./output/cfg/filebeat-prom.yml --path.data  ./filebeat-prom --path.home ./ --path.logs ./log/mlog/*
:: start "promtail" /min ./output/bin/win/promtail.exe -config.file=./output/cfg/promtail.yaml
:: start "filebeat-datalog" /min ./output/bin/win/filebeat-datalog.exe -e -c ./output/cfg/filebeat-datalog.yml --path.data  ./filebeat-datalog --path.home ./

:: ****************** lobby ******************
ping 127.0.0.1 -n 1
if %Debug% EQU  "l" (
    set  LOBBY_GRPC_PORT=50001
    echo ">>>>>>>>>>>>>>lobby daprd:"!LOBBY_GRPC_PORT!
    start "daprd" %BIN_PATH%\daprd.exe -config %DAPR_CONFIG% -app-id %LOBBY_APPID% -app-protocol %PROTOCOL% ^
        -app-port %LOBBY_APP_PORT% -dapr-http-port %LOBBY_HTTP_PORT% -dapr-grpc-port !LOBBY_GRPC_PORT! ^
        -components-path %COMPONENT_PATH% -enable-metrics false
 ) ELSE (
   echo ">>>>>>>>>>>>>>lobby"
    start %LOBBY_APPID% /min %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %LOBBY_APPID% ^
    -P %PROTOCOL% -p %LOBBY_APP_PORT% --dapr-http-port %LOBBY_HTTP_PORT% --dapr-grpc-port %LOBBY_GRPC_PORT% ^
    --dapr-http-read-buffer-size 4096 --log-level debug -d %COMPONENT_PATH% ^
    %BIN_PATH%\lobbyserver.exe appid=%LOBBY_APPID% inaddr=%LOBBY_APP_PORT% gport=%LOBBY_GRPC_PORT% pprof=%LOBBY_PPROF_PORT% dev=1 ^
    rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%
   )

:: ****************** actor ******************
ping 127.0.0.1 -n 1
if %Debug% EQU  "a" (
SETLOCAL
    set  ACTOR_GRPC_PORT=50001
    echo ">>>>>>>>>>>>>>actor daprd:"!ACTOR_GRPC_PORT!
    start "daprd" %BIN_PATH%\daprd.exe -config %DAPR_CONFIG% -app-id %ACTOR_APPID% -app-protocol http ^
        -app-port %ACTOR_APP_PORT% -dapr-http-port %ACTOR_HTTP_PORT% -dapr-grpc-port !ACTOR_GRPC_PORT! ^
        -placement-host-address localhost:6050 -components-path %COMPONENT_PATH% -enable-metrics false
ENDLOCAL
   ) ELSE (
   echo ">>>>>>>>>>>>>>actor"
   start %ACTOR_APPID% %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %ACTOR_APPID% ^
    -P http -p %ACTOR_APP_PORT% --dapr-http-port %ACTOR_HTTP_PORT% --dapr-grpc-port %ACTOR_GRPC_PORT% ^
    --dapr-http-read-buffer-size 4096 --log-level debug -d %COMPONENT_PATH% ^
    %BIN_PATH%\actorserver.exe appid=%ACTOR_APPID% actor=%ACTORS% inaddr=%ACTOR_APP_PORT% gport=%ACTOR_GRPC_PORT% pprof=%ACTOR_PPROF_PORT% dev=1 ^
    rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%
   )

:: ****************** login ******************
ping 127.0.0.1 -n 1
if %Debug% EQU  "lo" (
    set  LOGIN_GRPC_PORT=50001
    echo ">>>>>>>>>>>>>>login daprd:"!LOGIN_GRPC_PORT!
    start "daprd" %BIN_PATH%\daprd.exe -config %DAPR_CONFIG% -app-id %LOGIN_APPID% -app-protocol %PROTOCOL% ^
        -app-port %LOGIN_APP_PORT% -dapr-http-port %LOGIN_HTTP_PORT% -dapr-grpc-port !LOGIN_GRPC_PORT! ^
        -components-path %COMPONENT_PATH% -enable-metrics false
   ) ELSE (
   echo ">>>>>>>>>>>>>>login"
    start %LOGIN_APPID% /min %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %LOGIN_APPID% ^
    -P %PROTOCOL% -p %LOGIN_APP_PORT% --dapr-http-port %LOGIN_HTTP_PORT% --dapr-grpc-port %LOGIN_GRPC_PORT% ^
    --dapr-http-read-buffer-size 4096 --log-level debug -d %COMPONENT_PATH% ^
    %BIN_PATH%\loginserver.exe appid=%LOGIN_APPID% inaddr=%LOGIN_APP_PORT% outaddr=%LOGIN_OUT_PORT% gport=%LOGIN_GRPC_PORT% pprof=%LOGIN_PPROF_PORT% dev=1 ^
    rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%
   )

:: ****************** gate ******************
ping 127.0.0.1 -n 1
if %Debug% EQU  "g" (
    set  GATE_GRPC_PORT=50001
    echo ">>>>>>>>>>>>>>gate daprd:"!GATE_GRPC_PORT!
    echo ">>>>>>>>>>>>>> NEED DO: Manually set the gate name"
    start "daprd" %BIN_PATH%\daprd.exe -config %DAPR_CONFIG% -app-id %GATE_APPID% -app-protocol %PROTOCOL% ^
        -app-port %GATE_APP_PORT% -dapr-http-port %GATE_HTTP_PORT% -dapr-grpc-port !GATE_GRPC_PORT! ^
        -components-path %COMPONENT_PATH% -enable-metrics false
   ) ELSE (
   echo ">>>>>>>>>>>>>>gate"
   start %GATE_APPID% %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %GATE_APPID% ^
   -P %PROTOCOL% -p %GATE_APP_PORT% --dapr-http-port %GATE_HTTP_PORT% --dapr-grpc-port %GATE_GRPC_PORT% ^
   --dapr-http-read-buffer-size 4096 --log-level debug -d %COMPONENT_PATH% ^
   %BIN_PATH%\gateserver.exe appid=%GATE_APPID% inaddr=%GATE_APP_PORT% outaddr=%GATE_OUT_PORT% gport=%GATE_GRPC_PORT% pprof=%GATE_PPROF_PORT% dev=1 ^
   rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%
   )

:: ****************** bill ******************
ping 127.0.0.1 -n 1
if %Debug% EQU  "b" (
    set  BILL_GRPC_PORT=50001
    echo ">>>>>>>>>>>>>>bill daprd:"!BILL_GRPC_PORT!
    echo ">>>>>>>>>>>>>> NEED DO: Manually set the bill name"
    start "daprd" %BIN_PATH%\daprd.exe -config %DAPR_CONFIG% -app-id %BILL_APPID% -app-protocol %PROTOCOL% ^
        -app-port %BILL_APP_PORT% -dapr-http-port %BILL_HTTP_PORT% -dapr-grpc-port !BILL_GRPC_PORT! ^
        -components-path %COMPONENT_PATH% -enable-metrics false
   ) ELSE (
   echo ">>>>>>>>>>>>>>bill"
   start %BILL_APPID% /min %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %BILL_APPID% ^
   -P %PROTOCOL% -p %BILL_APP_PORT% --dapr-http-port %BILL_HTTP_PORT% --dapr-grpc-port %BILL_GRPC_PORT% ^
   --dapr-http-read-buffer-size 4096 --log-level debug -d %COMPONENT_PATH% ^
   %BIN_PATH%\billserver.exe appid=%BILL_APPID% inaddr=%BILL_APP_PORT% gport=%BILL_GRPC_PORT% webaddr=18001 pprof=%BILL_PPROF_PORT% dev=1 ^
    rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%
   )

:: ****************** idip ******************
ping 127.0.0.1 -n 1
if %Debug% EQU  "i" (
    set  IDIP_GRPC_PORT=50001
    echo ">>>>>>>>>>>>>>bill daprd:"!IDIP_GRPC_PORT!
    echo ">>>>>>>>>>>>>> NEED DO: Manually set the bill name"
    start "daprd" %BIN_PATH%\daprd.exe -config %DAPR_CONFIG% -app-id %IDIP_APPID% -app-protocol %PROTOCOL% ^
        -app-port %IDIP_APP_PORT% -dapr-http-port %IDIP_HTTP_PORT% -dapr-grpc-port !IDIP_GRPC_PORT! ^
        -components-path %COMPONENT_PATH% -enable-metrics false
   ) ELSE (
   echo ">>>>>>>>>>>>>>idip"
   start %IDIP_APPID% /min %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %IDIP_APPID% ^
   -P %PROTOCOL% -p %IDIP_APP_PORT% --dapr-http-port %IDIP_HTTP_PORT% --dapr-grpc-port %IDIP_GRPC_PORT% --log-level debug -d %COMPONENT_PATH% ^
   %BIN_PATH%\idipserver.exe appid=%IDIP_APPID% inaddr=%IDIP_APP_PORT% gport=%IDIP_GRPC_PORT% webaddr=19001 pprof=%IDIP_PPROF_PORT% dev=1 ^
    rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%
   )

:: ****************** guide ******************
ping 127.0.0.1 -n 1
if %Debug% EQU  "gu" (
    set  GUIDE_GRPC_PORT=50001
    echo ">>>>>>>>>>>>>>guide daprd:"!GUIDE_GRPC_PORT!
    echo ">>>>>>>>>>>>>> NEED DO: Manually set the bill name"
    start "daprd" %BIN_PATH%\daprd.exe -config %DAPR_CONFIG% -app-id %GUIDE_APPID% -app-protocol %PROTOCOL% ^
        -app-port %GUIDE_APP_PORT% -dapr-http-port %GUIDE_HTTP_PORT% -dapr-grpc-port !GUIDE_GRPC_PORT! ^
        -components-path %COMPONENT_PATH% -enable-metrics false
   ) ELSE (
   echo ">>>>>>>>>>>>>>guide"
   start %GUIDE_APPID% /min %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %GUIDE_APPID% ^
   -P %PROTOCOL% -p %GUIDE_APP_PORT% --dapr-http-port %GUIDE_HTTP_PORT% --dapr-grpc-port %GUIDE_GRPC_PORT% --log-level debug -d %COMPONENT_PATH% ^
   %BIN_PATH%\guideserver.exe appid=%GUIDE_APPID% inaddr=%GUIDE_APP_PORT% gport=%GUIDE_GRPC_PORT% pprof=%GUIDE_PPROF_PORT% dev=1 ^
   rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%
   )

:: ****************** battle ******************
echo ">>>>>>>>>>>>>>BATTLE"
    start %BATTLE_APPID% /min %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %BATTLE_APPID% ^
    -P http -p %BATTLE_APP_PORT% --dapr-http-port %BATTLE_HTTP_PORT% --dapr-grpc-port %BATTLE_GRPC_PORT% ^
    --dapr-http-read-buffer-size 4096 --log-level debug -d %COMPONENT_PATH% ^
    %BIN_PATH%\battleserver.exe dev=1 ^
    rdscfghost=%RDSCFGHOST% rdscfgpass=%RDSCFGPASS% rdscfgns=%RDSCFGNS% rdscfggroup=%RDSCFGGROUP%

ping 127.0.0.1 -n 1
rem start %MUSAE_APPID% %BIN_PATH%\dapr.exe run -a %MUSAE_APPID%  -P %PROTOCOL% -p 50009 --dapr-http-port 3509 -d %COMPONENT_PATH%  --log-level debug  %BIN_PATH%\musaectl.exe appid=%MUSAE_APPID%

:: ****************** musae ******************
:: ping 127.0.0.1 -n 1
:: start %MUSAE_APPID% %BIN_PATH%\dapr.exe run -c %DAPR_CONFIG% -a %MUSAE_APPID%  -d %COMPONENT_PATH%  --log-level debug --dapr-http-port 3500

:: start "filebeat-prom" /min ./output/bin/win/filebeat-prom.exe -e -c ./output/cfg/filebeat-prom.yml --path.data  ./filebeat-prom --path.home ./ --path.logs ./log/mlog/*
