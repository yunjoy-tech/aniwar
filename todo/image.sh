#!/usr/bin/env bash

#export DOCKER_REPO=192.168.2.8:5000
#export DOCKER_IMAGE_PREFIX=aniwar

# aniwar server
docker build --target aniwarserver \
  --build-arg GO_VERSION=${GO_VERSION} \
  --build-arg VERSION=${VERSION} \
  --tag aniwarserver:${DOCKER_TAG} \
  --file Dockerfile .

# docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/aniwarserver:${DOCKER_TAG}
docker tag aniwarserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/aniwarserver:${DOCKER_TAG}
docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/aniwarserver:${DOCKER_TAG}


## login server
#docker build --target loginserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag loginserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/loginserver:${DOCKER_TAG}
#docker tag loginserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/loginserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/loginserver:${DOCKER_TAG}
#
#
## gate server
#docker build --target gateserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag gateserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/gateserver:${DOCKER_TAG}
#docker tag gateserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/gateserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/gateserver:${DOCKER_TAG}
#
## actor server
#docker build --target actorserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag actorserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/actorserver:${DOCKER_TAG}
#docker tag actorserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/actorserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/actorserver:${DOCKER_TAG}
#
## lobby server
#docker build --target lobbyserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag lobbyserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/lobbyserver:${DOCKER_TAG}
#docker tag lobbyserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/lobbyserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/lobbyserver:${DOCKER_TAG}
#
## idip server
#docker build --target idipserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag idipserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/idipserver:${DOCKER_TAG}
#docker tag idipserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/idipserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/idipserver:${DOCKER_TAG}
#
## bill server
#docker build --target billserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag billserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/billserver:${DOCKER_TAG}
#docker tag billserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/billserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/billserver:${DOCKER_TAG}
#
## battle server
#docker build --target battleserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag battleserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/battleserver:${DOCKER_TAG}
#docker tag battleserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/battleserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/battleserver:${DOCKER_TAG}
#
## gm server
#docker build --target gmserver \
#  --build-arg GO_VERSION=${GO_VERSION} \
#  --build-arg VERSION=${VERSION} \
#  --tag gmserver:${DOCKER_TAG} \
#  --file Dockerfile .
#
## docker push  ${DOCKER_REPO}/${DOCKER_IMAGE_PREFIX}/gmserver:${DOCKER_TAG}
#docker tag gmserver:${DOCKER_TAG} tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/gmserver:${DOCKER_TAG}
#docker push tsh-aniwar-dev-cr-registry.cn-shanghai.cr.aliyuncs.com/aniwar/gmserver:${DOCKER_TAG}
