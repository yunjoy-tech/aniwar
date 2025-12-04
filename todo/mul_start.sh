#!/bin/bash

export BUILD_DISPLAY_NAME=$1

export JOB_BASE_NAME=$2

export PUBLIC=$3

export MAIN_VERSION=$4

export BUILD_NUMBER=$5

# 镜像
export IMAGE=$6
# jenkins 工程目录
export path_project=$7
# 分支
export CODE_BRANCH=$8
# 项目名字
export PROJECT_NAME=$9

export SUB_BRANCH=${10}
export CLIENT_RANCH=${11}
export COMMIT=${12}
export Restart=${13}

echo "参数:"$@
for i in $@; do
  echo $i
done

export VERSION=${MAIN_VERSION}.${BUILD_NUMBER}

function pull_project() {
  string1=$1
  string2=$2
  BUILD_NUMBER=$3
  path_project=$4

  ANIWAR_CLIENT_BRANCH=main
  ANIWAR_BRANCH=develop
  MUSAE_BRANCH=main
  BATTLESERVER_BRANCH=main
  PROTOCOL_BRANCH=main
  SVN_ADDRESS=design/excelSource

  if [[ ${CODE_BRANCH} == release* ]]; then
    echo "匹配到release"
    ANIWAR_CLIENT_BRANCH=${CLIENT_RANCH}
    echo "匹配到release 客户端分支" ${ANIWAR_CLIENT_BRANCH}
    ANIWAR_BRANCH=${CODE_BRANCH}
    MUSAE_BRANCH=${CODE_BRANCH}
    BATTLESERVER_BRANCH=${CODE_BRANCH}
    PROTOCOL_BRANCH=${CODE_BRANCH}
    SVN_ADDRESS=design/branch/${SUB_BRANCH}
  fi

  console="http://192.168.2.8:8089/job/$string2/${string1:1}/console"
  python3 /work/script/jenkinshook_server.py --state 3 --num ${BUILD_NUMBER} --console $console --title ${PROJECT_NAME}版本开始构建

  source /etc/profile
  #echo $PATH
  whoami
  source ~/.bash_profile
  env
  go env

  CLIENT_PATH=${path_project}/aniwar-client
  echo "aniwar-client 分支:" ${ANIWAR_CLIENT_BRANCH}
  if [ ! -d $CLIENT_PATH ]; then
    git clone -b ${ANIWAR_CLIENT_BRANCH} https://gitlab.musadisca-games.com/aniwar/aniwar-client.git
  else
    cd $CLIENT_PATH
    git checkout ${ANIWAR_CLIENT_BRANCH}
    git checkout -- .
    git pull origin ${ANIWAR_CLIENT_BRANCH}
    cd -
  fi

  #ANIWAR_PATH=./aniwar
  ANIWAR_PATH=${path_project}/aniwar
  echo "aniwar 分支:" ${ANIWAR_BRANCH}
  if [ ! -d $ANIWAR_PATH ]; then
    git clone -b ${ANIWAR_BRANCH} https://gitlab.musadisca-games.com/server/aniwar.git
  else
    cd $ANIWAR_PATH
    git checkout ${ANIWAR_BRANCH}
    git checkout -- .
    git pull origin ${ANIWAR_BRANCH}
    cd -
  fi

  #MUSAE_PATH=./musae
  echo "musae 分支:" ${MUSAE_BRANCH}
  MUSAE_PATH=${path_project}/musae
  if [ ! -d $MUSAE_PATH ]; then
    git clone -b ${MUSAE_BRANCH} https://gitlab.musadisca-games.com/server/musae.git
  else
    cd $MUSAE_PATH
    git checkout ${MUSAE_BRANCH}
    git checkout -- .
    git pull origin ${MUSAE_BRANCH}
    cd -
  fi

  #CHECKBATTLE_PATH=./aniwar/src/battleserver#
  echo "battleserver 分支:" ${BATTLESERVER_BRANCH}
  CHECKBATTLE_PATH=${path_project}/aniwar/src/battleserver
  if [ ! -d $CHECKBATTLE_PATH ]; then
    cd ${path_project}/aniwar/src
    git clone -b ${BATTLESERVER_BRANCH} https://gitlab.musadisca-games.com/common/battleserver.git
    cd -
  else
    cd $CHECKBATTLE_PATH
    git checkout ${BATTLESERVER_BRANCH}
    git checkout -- .
    git pull origin ${BATTLESERVER_BRANCH}
    cd -
  fi

  #PROTO_PATH=./aniwar/src/proto/protocol
  echo "protocol 分支:" ${PROTOCOL_BRANCH}
  PROTO_PATH=${path_project}/aniwar/src/proto/protocol
  if [ ! -d $PROTO_PATH ]; then
    cd ${path_project}/aniwar/src/proto
    git clone -b ${PROTOCOL_BRANCH} https://gitlab.musadisca-games.com/common/protocol.git
    cd -
  else
    cd $PROTO_PATH
    git checkout ${PROTOCOL_BRANCH}
    git checkout -- .
    git pull origin ${PROTOCOL_BRANCH}
    cd -
  fi

  #SVN_PATH=./design/excelSource
  SVN_PATH=${path_project}/design/excelSource
  rm -rf $SVN_PATH
  if [ ! -d $SVN_PATH ]; then
    mkdir -p $SVN_PATH
    echo "svn address:" svn://192.168.2.15/${SVN_ADDRESS}
    svn co svn://192.168.2.15/${SVN_ADDRESS} $SVN_PATH
  else
    svn revert -R $SVN_PATH
    svn up $SVN_PATH
  fi

}
function res() {
  path_project=$1
  project_name=$2
  version=$3
  docker_tag=$4
  cd ${path_project}/aniwar
  echo "make start"
  echo "make res"
  source ~/.bash_profile
  pwd

  rm -rf ./src/excel/data/*
  mkdir -m 777 -p ./src/excel/data
  rm -rf ./output/data/*
  mkdir -m 777 -p ./output/data

  chmod +x ./src/proto/protoc
  chmod +x ./src/proto/protoc-gen-go
  chmod +x ./src/proto/protoc-gen-go-grpc

  echo "make auto-export-excel-tool"
  cd ./tools/auto-export-excel-tool
  chmod +x ./protoc
  chmod +x ./protoc-gen-go
  chmod +x ./protoc-gen-go-grpc
  #bash ./install.sh
  #bash ./go-build.sh
  bash ./start.sh
  cd -

  echo "make proto"
  cd ./src/proto
  #go mod tidy
  bash ./build.sh
  cd -

  #make res

  #更新battleserver代码和资源
  make battleCopyFromClient

  echo "make linux"
  #  if [[ $project_name == "release" ]]; then
  #    make linux_release
  #  else
  #    make linux_develop
  #  fi
  ./make.sh linux "${project_name}" "${version}" "${docker_tag}"
  makeRet=$?
  if [ $makeRet -ne 0 ]; then
    echo "====>>>>" $comment "exce 'make linux' got fail, all task stop!!!"
    exit $makeRet # end the script running
  fi

  echo "make win"
  #  if [[ $project_name == "release" ]]; then
  #    make win_release
  #  else
  #    make win_develop
  #  fi
  ./make.sh win "${project_name}" "${version}" "${docker_tag}"
  echo "make end"
  cd -
}
function svn_commit() {
  path_project=$1
  COMMIT=$2
  echo "提交SVN" ${COMMIT}
  source ~/.bash_profile
  SVN_SRV_PATH=/data/svn/server/${COMMIT}
  rm -rf $SVN_SRV_PATH

  if [ ! -d $SVN_SRV_PATH ]; then
    mkdir -m 755 -p $SVN_SRV_PATH
    cd $SVN_SRV_PATH
    svn co svn://192.168.2.15/server/${COMMIT} .
  else
    cd $SVN_SRV_PATH
    #chown -R root:root .
    #svn revert -R .
    svn up .
  fi
  cd -

  cd ${path_project}/aniwar
  /bin/cp -rf ./output $SVN_SRV_PATH/
  /bin/cp -rf ./script $SVN_SRV_PATH/
  /bin/cp -rf ./*.sh $SVN_SRV_PATH/
  /bin/cp -rf ./*.bat $SVN_SRV_PATH/
  /bin/cp -rf ./Makefile $SVN_SRV_PATH/
  /bin/cp -rf ./localServer-README.txt $SVN_SRV_PATH/

  TOOL_PATH=$SVN_SRV_PATH/tools
  if [ ! -d $TOOL_PATH ]; then
    mkdir -p $TOOL_PATH
  fi

  /bin/cp -rf ./tools/auto-export-excel-tool $TOOL_PATH/
  /bin/cp -rf ./tools/svn $TOOL_PATH/

  cd $SVN_SRV_PATH
  # unix2dos start.bat stop.bat res.bat restart.bat pull.bat push.bat proto.bat
  svn propset svn:eol-style CRLF start.bat stop.bat res.bat restart.bat pull.bat push.bat proto.bat
  svn add . --no-ignore --force
  svn commit -m "#b"${BUILD_TAG}

}

function update_service() {

  path_project=$1
  server_version=$2
  restart=$3
  source ~/.bash_profile
  source ~/.bashrc
  cd ${path_project}/aniwar
  pwd
  #启动 2.8服务器
  rm -rf /data/server/output/*
  #mkdir -m 777 -p /data/server/log/plog

  /bin/cp -rf ./output /mnt/nas/k3s/dev/app/server/${server_version}/

  if ${restart}; then
    # 触发热更
    echo "update service 触发热更"
    case ${PROJECT_NAME} in
    release)
      echo "release 触发 http://192.168.2.11:29001/api/hotReload 热更"
      echo "release 触发 http://192.168.2.11:24001/api/hotReload 热更"
      #curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.11:29001/api/reload"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.11:29001/api/hotReload"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.11:24001/api/hotReload"
      ;;
    develop)

      echo "develop 触发 http://192.168.2.8:29001/api/hotReload 热更"
      echo "develop 触发 http://192.168.2.8:24001/api/hotReload 热更"
      #curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.8:29001/api/reload"
      #curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.10:29001/api/reload"

      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.8:29001/api/hotReload"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.8:24001/api/hotReload"
      echo "develop 触发 http://192.168.2.10:29001/api/hotReload 热更"
      echo "develop 触发 http://192.168.2.10:24001/api/hotReload 热更"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.10:29001/api/hotReload"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.10:24001/api/hotReload"
      ;;
    excel_compile)
      echo "excel_compile 触发 http://192.168.2.8:29001/api/hotReload 热更"
      echo "excel_compile 触发 http://192.168.2.8:24001/api/hotReload 热更"
      #curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.8:29001/api/reload"
      #curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.10:29001/api/reload"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.8:29001/api/hotReload"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.8:24001/api/hotReload"

      echo "excel_compile 触发 http://192.168.2.10:29001/api/hotReload 热更"
      echo "excel_compile 触发 http://192.168.2.10:24001/api/hotReload 热更"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.10:29001/api/hotReload"
      curl -H "Content-Type: application/json" -X POST -d '{"type": "excel", "files":"all" }' "http://192.168.2.10:24001/api/hotReload"

      ;;

    *)
      echo "system error !!!"
      ;;
    esac

  else
    echo "update service 重启服务"
    # shellcheck disable=SC2164
    cd /root/ops/ops-chart/scripts
    ./helm-install-dev.sh release dev 1.0.0
    # shellcheck disable=SC2164
    cd -

    case ${PROJECT_NAME} in
    release)
      echo "release 触发 http://192.168.2.11:8089/ UpdateServer"
      /usr/bin/java -jar /data/jenkins-cli.jar -s http://192.168.2.11:8089/ -auth admin:aniwar.1024 build UpdateServer -p TYPE="服务器"
      ;;
    develop)
      echo "develop 触发 http://192.168.2.10:8089/ UpdateServer"
      /usr/bin/java -jar /data/jenkins-cli.jar -s http://192.168.2.10:8089/ -auth admin:aniwar.1024 build UpdateServer -p TYPE="服务器"
      ;;
    excel_compile) ;;
    *)
      echo "system error !!!"
      ;;
    esac
  fi
  ps -ef | grep output

  cd -
}

function tar_version() {
  PUBLIC=$1
  VERSION=$2
  IMAGE=$3
  path_project=$4

  cd ${path_project}/aniwar

  if ${PUBLIC}; then
    chmod +x ./output/bin/linux/versionTool
    #VERSION=${MAIN_VERSION}.${BUILD_NUMBER}
    pkgName="aniwar"-${PROJECT_NAME}-${VERSION}-${IMAGE}-public-$(date "+%Y_%m_%d_%H_%M_%S")
    echo "pkg name:" ${pkgName}
    fileName=${pkgName}.tar.gz
    uploadName=upload-${pkgName}.sh
    versionDir=./pkg/${pkgName}
    mkdir -m 755 -p ${versionDir}

    echo "
    #!/usr/bin/env bash
    scp ./${fileName} tsh-aniwar-dev-jump-bastion:/home/aniwar/pkg/${fileName}
    scp ./${installName} tsh-aniwar-dev-jump-bastion:/home/aniwar/pkg/${installName}
    " >./pkg/${pkgName}/${uploadName}

    ./output/bin/linux/versionTool build --version ${VERSION} --mode server \
      --file ./versionfiles.txt --output ${versionDir}/${fileName} \
      --ignore ./versionignore.txt

    # tar -zxvf ${versionDir}/${fileName} -C /data/nfs/version/server/version/
    mv ${versionDir} /data/nfs/version/server/backup/${pkgName}
    # 更新version list
    echo ${pkgName} >>/data/nfs/version/server/list.txt

    echo "发布版本: "${fileName}
    #/usr/bin/cp -rf ${versionDir} /data/server-version/
    #cd /data/server-version/
    #echo `pwd`
    #git restore .
    #git add .
    #git commit -m "upload version: "${fileName}
    #git push
    redis-cli -h 192.168.2.7 SADD version:server ${VERSION}
  fi
  cd -
}

function gmt() {
  # 更新gm工具
  source ~/.bash_profile
  cd ${path_project}/aniwar
  pwd

  PROC_NAME=gmserver
  ProcNumber=$(ps -ef | grep -w $PROC_NAME | grep -v grep | wc -l)
  if [ $ProcNumber -le 0 ]; then
    BUILD_ID=dontKillMe
    #make gmstart
    cd output/gm && chmod +x gmserver && bash ./start.sh
  else
    make gmstop
    sleep 2
    BUILD_ID=dontKillMe
    #make gmstart
    cd output/gm && chmod +x gmserver && bash ./start.sh
  fi
  ps -ef | grep gmserver

  cd -
}

echo "############################      start pull project!!!      ############################"
pull_project ${BUILD_DISPLAY_NAME} ${JOB_BASE_NAME} ${BUILD_NUMBER} ${path_project}
echo "############################  pull project successful !!!    ############################"

echo "############################     start make  res!!!          ############################"
res ${path_project} ${JOB_BASE_NAME} ${VERSION} ${IMAGE}
echo "############################ start make res successful !!!   ############################"

#echo "############################     start svn commit !!!        ############################"
#svn_commit ${path_project}
#echo "############################     svn commit successful !!!   ############################"

case ${PROJECT_NAME} in
gmt)
  echo "更新gm工具"
  echo "############################        update gm  tools !!!    ############################"
  gmt
  echo "############################   update gm tools successful!! ############################"
  ;;
dev_test)
  echo "dev_test"
  echo "############################        start service !!!       ############################"
  update_service ${path_project} 1.0.0 ${Restart}
  echo "############################   start service successful!!!  ############################"

  echo "############################   start tar version  !!!       ############################"
  tar_version ${PUBLIC} ${VERSION} ${IMAGE} ${path_project}
  echo "############################   tar version  successful!!!   ############################"
  ;;
develop)
  echo "develop"
  echo "############################        start service !!!       ############################"
  update_service ${path_project} 1.0.0 ${Restart}
  echo "############################   start service successful!!!   ############################"

  echo "############################     start svn commit !!!        ############################"
  svn_commit ${path_project} debug
  echo "############################     svn commit successful !!!   ############################"

  echo "############################   start tar version  !!!   ############################"
  tar_version ${PUBLIC} ${VERSION} ${IMAGE} ${path_project}
  echo "############################   tar version  successful!!! ############################"
  ;;
release)
  echo "release"
  echo "############################        start service !!!       ############################"
  update_service ${path_project} 2.0.0 ${Restart}
  echo "############################   start service successful!!!   ############################"

  echo "############################     start svn commit !!!        ############################"
  svn_commit ${path_project} release
  echo "############################     svn commit successful !!!   ############################"

  echo "############################   start tar version  !!!   ############################"
  tar_version ${PUBLIC} ${VERSION} ${IMAGE} ${path_project}
  echo "############################   tar version  successful!!! ############################"
  ;;
excel_compile)
  echo "excel_compile"
  echo "############################        start service !!!       ############################"
  update_service ${path_project} 1.0.0 true
  echo "############################   start service successful!!!   ############################"

  echo "############################     start svn commit !!!        ############################"
  svn_commit ${path_project} debug
  echo "############################     svn commit successful !!!   ############################"

  echo "############################   excel_compile  successful!!! ############################"
  ;;
*)
  echo "system error !!!"
  ;;
esac
