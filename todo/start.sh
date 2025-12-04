#!/usr/bin/env bash

if [ $# -eq 0 ];
then
    echo 'please set platform param, ps start.sh [win/linux/mac], default linux'
    #exit
fi

CGO_ENABLED=0
PLAT_FORM="linux"
EXT=""
# shellcheck disable=SC1073
if [ $1 == "win" ]; then
    PLAT_FORM="win"
    EXT=".exe"
elif [ $1 == "linux" ]; then
    PLAT_FORM="linux"
else
    PLAT_FORM="mac"
fi

LOGIN_APPID="login"

local_ip=`ifconfig -a|grep inet|grep 192.168.|grep -v inet6|awk '{print $2}'|tr -d "addr:"`

GUIDE_APPID="guide"
GATE_APPID="gate_${local_ip//./_}"
LOBBY_APPID="lobby"
ACTOR_APPID="actor"
BATTLE_APPID="battle"
BILL_APPID="bill"
IDIP_APPID="idip"
ROBOT_APPID="robot"
MUSAE_APPID="musae"

PROTOCOL="http"

GUIDE_APP_PORT=20001
GUIDE_HTTP_PORT=20002
GUIDE_GRPC_PORT=20003
GUIDE_PPROF_PORT=20004
GUIDE_METRIC_PORT=20005

LOGIN_OUT_PORT=12001
LOGIN_APP_PORT=21001
LOGIN_HTTP_PORT=21002
LOGIN_GRPC_PORT=21003
LOGIN_PPROF_PORT=21004
LOGIN_METRIC_PORT=21005

# GATE_OUT_PORT和server.conf中gatePort参数保持一致
GATE_OUT_PORT=13001
GATE_APP_PORT=22001
GATE_HTTP_PORT=22002
GATE_GRPC_PORT=22003
GATE_PPROF_PORT=22004
GATE_METRIC_PORT=22005

LOBBY_APP_PORT=23001
LOBBY_HTTP_PORT=23002
LOBBY_GRPC_PORT=23003
LOBBY_PPROF_PORT=23004
LOBBY_METRIC_PORT=23005

ACTOR_APP_PORT=24001
ACTOR_HTTP_PORT=24002
ACTOR_GRPC_PORT=24003
ACTOR_PPROF_PORT=24004
ACTOR_METRIC_PORT=24005

ACTOR2_APP_PORT=24011
ACTOR2_HTTP_PORT=24012
ACTOR2_GRPC_PORT=24013
ACTOR2_PPROF_PORT=24014
ACTOR2_METRIC_PORT=24015

MAIL_APP_PORT=25001
MAIL_HTTP_PORT=25002
MAIL_GRPC_PORT=25003
MAIL_PPROF_PORT=25004
MAIL_METRIC_PORT=25005

MAIL2_APP_PORT=25011
MAIL2_HTTP_PORT=25012
MAIL2_GRPC_PORT=25013
MAIL2_PPROF_PORT=25014
MAIL2_METRIC_PORT=25015

BATTLE_APP_PORT=27001
BATTLE_HTTP_PORT=27002
BATTLE_GRPC_PORT=27003
BATTLE_PPROF_PORT=27004
BATTLE_METRIC_PORT=27005

BILL_APP_PORT=28001
BILL_HTTP_PORT=28002
BILL_GRPC_PORT=28003
BILL_PPROF_PORT=28004
BILL_METRIC_PORT=28005

IDIP_APP_PORT=29001
IDIP_HTTP_PORT=29002
IDIP_GRPC_PORT=29003
IDIP_PPROF_PORT=29004
IDIP_METRIC_PORT=29005

BIN_PATH=./output/bin/$PLAT_FORM
COMPONENT_PATH=./output/cfg/component
DAPR_CONFIG=./output/cfg/dapr-config.yaml

echo "bin path: "$BIN_PATH

ACTORS="UserActor|RoomActor|AllianceActor|CenterActor|MailActor"
RDSCFGHOST=aniwar-dev-global.redis.rds.aliyuncs.com:36379
RDSCFGPASS=0XjLawDmTfo5IVeP
RDSCFGNS=cn
RDSCFGGROUP=dev

chmod +x $BIN_PATH/consul
chmod +x $BIN_PATH/placement
#chmod +x $BIN_PATH/promtail

nohup  $BIN_PATH/consul agent -dev -ui -client 0.0.0.0 >./log/consul.log 2>&1 &

nohup  $BIN_PATH/placement -port 50005 -metrics-port 9091 >./log/placement.log 2>&1 &

# nohup $BIN_PATH/promtail -config.file=./output/cfg/promtail.yaml >./log/plog/promtail.log 2>&1 &

sleep 0
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${ACTOR_APPID} -P ${PROTOCOL} --app-port $ACTOR_APP_PORT --dapr-http-port $ACTOR_HTTP_PORT --dapr-grpc-port $ACTOR_GRPC_PORT \
  --metrics-port $ACTOR_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/actorserver \
  appid=${ACTOR_APPID} actor=${ACTORS} inaddr=$ACTOR_APP_PORT gport=$ACTOR_GRPC_PORT pprof=$ACTOR_PPROF_PORT dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &

sleep 0
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${ACTOR_APPID} -P http --app-port $ACTOR2_APP_PORT --dapr-http-port $ACTOR2_HTTP_PORT --dapr-grpc-port $ACTOR2_GRPC_PORT \
  --metrics-port $ACTOR2_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/actorserver \
  appid=${ACTOR_APPID} actor=${ACTORS} inaddr=$ACTOR2_APP_PORT gport=$ACTOR2_GRPC_PORT pprof=$ACTOR2_PPROF_PORT dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &

sleep 5
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${IDIP_APPID} -P ${PROTOCOL} --app-port $IDIP_APP_PORT --dapr-http-port $IDIP_HTTP_PORT --dapr-grpc-port $IDIP_GRPC_PORT \
  --metrics-port $IDIP_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/idipserver \
   appid=${IDIP_APPID} inaddr=$IDIP_APP_PORT gport=$IDIP_GRPC_PORT pprof=$IDIP_PPROF_PORT webaddr=19001 dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &

sleep 0
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${LOGIN_APPID} -P ${PROTOCOL} --app-port $LOGIN_APP_PORT --dapr-http-port $LOGIN_HTTP_PORT --dapr-grpc-port $LOGIN_GRPC_PORT \
  --metrics-port $LOGIN_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/loginserver \
  appid=${LOGIN_APPID} inaddr=$LOGIN_APP_PORT gport=$LOGIN_GRPC_PORT pprof=$LOGIN_PPROF_PORT dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &

sleep 0
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${GATE_APPID} -P ${PROTOCOL} --app-port $GATE_APP_PORT --dapr-http-port $GATE_HTTP_PORT --dapr-grpc-port $GATE_GRPC_PORT \
  --metrics-port $GATE_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/gateserver \
  appid=${GATE_APPID} inaddr=$GATE_APP_PORT outaddr=$GATE_OUT_PORT gport=$GATE_GRPC_PORT pprof=$GATE_PPROF_PORT dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &

sleep 0
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${GUIDE_APPID} -P ${PROTOCOL} --app-port $GUIDE_APP_PORT --dapr-http-port $GUIDE_HTTP_PORT --dapr-grpc-port $GUIDE_GRPC_PORT \
  --metrics-port $GUIDE_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/guideserver \
  appid=${GUIDE_APPID} inaddr=$GUIDE_APP_PORT gport=$GUIDE_GRPC_PORT pprof=$GUIDE_PPROF_PORT dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &

sleep 0
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${BILL_APPID} -P ${PROTOCOL} --app-port $BILL_APP_PORT --dapr-http-port $BILL_HTTP_PORT --dapr-grpc-port $BILL_GRPC_PORT \
  --metrics-port $BILL_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/billserver \
  appid=${BILL_APPID} inaddr=$BILL_APP_PORT gport=$BILL_GRPC_PORT pprof=$BILL_PPROF_PORT dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &



sleep 0
nohup $BIN_PATH/dapr run -c ${DAPR_CONFIG} -a ${BATTLE_APPID} -P ${PROTOCOL} --app-port $BATTLE_APP_PORT --dapr-http-port $BATTLE_HTTP_PORT --dapr-grpc-port $BATTLE_GRPC_PORT \
  --metrics-port $BATTLE_METRIC_PORT --log-level debug -d $COMPONENT_PATH $BIN_PATH/battleserver \
  appid=${BATTLE_APPID} inaddr=$BATTLE_APP_PORT gport=$BATTLE_GRPC_PORT pprof=$BATTLE_PPROF_PORT dev=1 \
  rdscfghost=${RDSCFGHOST} rdscfgpass=${RDSCFGPASS} rdscfgns=${RDSCFGNS} rdscfggroup=${RDSCFGGROUP} \
  >/dev/null 2>&1 &


sleep 0
# nohup python3 script/logmonitor.py >/dev/null 2>&1 &

