#!/usr/bin/env bash

export VERSION=$3
export DOCKER_TAG=$4

function checkFuncSucc() {
  local ret=$1
  local comment=$2

  if [ $ret != 0 ]; then
    echo "====>>>>" $comment "got fail"
    exit $ret # end the script running
  fi
  echo "====>>>>" $comment "succeed"
}

if [ $# -eq 0 ]; then
  echo 'please set platform param, ps make.sh [win/linux/mac], default linux'
  #exit
fi

export CGO_ENABLED=0
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
  sh res.sh
fi

gcflagsParams="-N -l -m"
if [[ $2 == "release" ]]; then
  gcflagsParams=""
fi

#go env -w GO111MODULE=on
#go env -w GOPROXY=https://goproxy.cn,https://gitlab.musadisca-games.com/wangxw,direct
#go env -w GOPRIVATE="*.musadisca-games.com"
#go env -w GONOSUMDB=gitlab.musadisca-games.com
#GIT_TERMINAL_PROMPT=1 go get -u  gitee.com/aniwar2/musae@main

#RACE="-race"
RACE=""
GoVersion=$(go version)
BuildTime=$(date +%Y-%m-%d_%H_%M_%S)
BuildVersion=""
#BuildVersion=`git tag --sort=committerdate | tail -n 1`
#CommitID=`git rev-parse HEAD`
if [[ -n "${GIT_HASH}" ]]; then
  BuildVersion="${GIT_HASH}"
else
  BuildVersion=$(git rev-parse HEAD)
fi

echo ${GoVersion}
LDFLAGS=-ldflags="-X gitee.com/aniwar2/musae/global.APP_VERSION=${VERSION}|${DOCKER_TAG}|${BuildVersion}|${BuildTime} -X gitee.com/aniwar2/musae/global.VERSION=${VERSION}"
if [[ $2 == "release" ]]; then
  LDFLAGS=-ldflags="-X gitee.com/aniwar2/musae/global.APP_VERSION=${VERSION}|${DOCKER_TAG}|${BuildVersion}|${BuildTime} -X gitee.com/aniwar2/musae/global.VERSION=${VERSION} -w"
fi

echo ${LDFLAGS}
echo "gcflags参数:" "${gcflagsParams}"

if [ $1 == "win" ]; then
  export GOOS=windows
  export GOARCH=amd64
elif [ $1 == "linux" ]; then
  export GOOS=linux
  export GOARCH=amd64
elif [ $1 == "linux" ]; then
  export GOOS=darwin
  export GOARCH=amd64
fi
echo "${GoVersion}"

echo "start build"
go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/guideserver"$EXT ./src/guideserver
checkFuncSucc $? "build guideserver"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/loginserver"$EXT ./src/loginserver
checkFuncSucc $? "build loginserver"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/gateserver"$EXT ./src/gateserver
checkFuncSucc $? "build gateserver"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/lobbyserver"$EXT ./src/lobbyserver
checkFuncSucc $? "build lobbyserver"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/actorserver"$EXT ./src/actorserver
checkFuncSucc $? "build actorserver"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/billserver"$EXT ./src/billserver
checkFuncSucc $? "build billserver"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/idipserver"$EXT ./src/idipserver
checkFuncSucc $? "build idipserver"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM"/robot"$EXT ./src/robot
#checkFuncSucc $? "build robot"

go build -gcflags "${gcflagsParams}" "${LDFLAGS}" ${RACE} -o "./output/bin/"$PLAT_FORM'/musaectl'$EXT ./tools/musaectl
#checkFuncSucc $? "build musaectl"

if [ $1 == "win" ]; then
  # update
  sh battleCopyFromClient.sh
  # build
  dotnet publish -c Release -r win-x64 --self-contained true -p:PublishReadyToRun=false /p:PublishSingleFile=true -o output/bin/$PLAT_FORM ./src/battleserver
  checkFuncSucc $? "build battleserver"
elif [ $1 == "linux" ]; then
  # update
  sh battleCopyFromClient.sh
  # build
  dotnet publish -c Release -r linux-x64 --self-contained true -p:PublishReadyToRun=true /p:PublishSingleFile=true -o output/bin/$PLAT_FORM ./src/battleserver
  checkFuncSucc $? "build battleserver"
elif [ $1 == "mac" ]; then
    # update
    sh battleCopyFromClient.sh
    # build
    dotnet publish -c Release -r osx.13-x64 --self-contained true -p:PublishReadyToRun=false /p:PublishSingleFile=true -o output/bin/$PLAT_FORM ./src/battleserver
    checkFuncSucc $? "build battleserver"
fi

echo "ALL SERVERs build success!"
