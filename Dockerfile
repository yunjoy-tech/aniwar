
#
# BUILD ENVIRONMENT
# -----------------
#ARG GO_VERSION
#FROM golang:${GO_VERSION} as builder

#WORKDIR /build
#COPY ./output ./output
# copy source
#COPY src/ src/
#COPY ./musae /app/musae
#COPY go.mod go.mod
#COPY go.sum go.sum
#RUN go mod download
#
#ENV CGO_ENABLED=0
#ENV GOOS=linux
#ENV GOARCH=amd64
#ENV GO111MODULE=on
#
#RUN go env -w GOPROXY=https://goproxy.cn,direct
#RUN go build -mod=readonly "-ldflags=-s -w" ./...
#
#ARG VERSION
#
#RUN go build -mod=readonly -o ./output/bin/loginserver ./src/loginserver
#RUN go build -mod=readonly -o ./output/bin/gateserver ./src/gateserver
#RUN go build -mod=readonly -o ./output/bin/lobbyserver ./src/lobbyserver
#RUN go build -mod=readonly -o ./output/bin/actorserver ./src/actorserver
#RUN go build -mod=readonly -o ./output/bin/idipserver ./src/idipserver


#
# IMAGE TARGETS
# -------------
#FROM alpine:3.17.0 as loginserver
#WORKDIR /app
#COPY  ./output/bin/linux/loginserver ./output/bin/linux/loginserver
#COPY  ./output/cfg ./output/cfg
#COPY  ./output/data ./output/data
#USER root:root
#ENTRYPOINT ["./output/bin/linux/loginserver"]

#FROM alpine:3.17.0 as gateserver
#WORKDIR /app
#COPY  ./output/bin/linux/gateserver ./output/bin/linux/gateserver
#COPY  ./output/cfg ./output/cfg
#COPY  ./output/data ./output/data
#USER root:root
#ENTRYPOINT ["./output/bin/linux/gateserver"]

#FROM alpine:3.17.0 as lobbyserver
#WORKDIR /app
#COPY  ./output/bin/linux/lobbyserver ./output/bin/linux/lobbyserver
#COPY  ./output/cfg ./output/cfg
#COPY  ./output/data ./output/data
#USER root:root
#ENTRYPOINT ["./output/bin/linux/lobbyserver"]

#FROM alpine:3.17.0 as actorserver
#WORKDIR /app
#COPY  ./output/bin/linux/actorserver ./output/bin/linux/actorserver
#COPY  ./output/cfg ./output/cfg
#COPY  ./output/data ./output/data
#USER root:root
#ENTRYPOINT ["./output/bin/linux/actorserver"]

#FROM alpine:3.17.0 as idipserver
#WORKDIR /app
#COPY  ./output/bin/linux/idipserver ./output/bin/linux/idipserver
#COPY  ./output/cfg ./output/cfg
#COPY  ./output/data ./output/data
#USER root:root
#ENTRYPOINT ["./output/bin/linux/idipserver"]

#FROM alpine:3.17.0 as billserver
#WORKDIR /app
#COPY  ./output/bin/linux/billserver ./output/bin/linux/billserver
#COPY  ./output/cfg ./output/cfg
#COPY  ./output/data ./output/data
#USER root:root
#ENTRYPOINT ["./output/bin/linux/billserver"]

#FROM alpine:3.17.0 as gmserver
#WORKDIR /app
#COPY  ./output/gm/gmserver ./output/gm/gmserver
#COPY  ./output/gm/config.yaml ./output/gm/config.yaml
#USER root:root
#ENTRYPOINT ["./output/gm/gmserver"]

#FROM alpine:3.17.0 as aniwarserver
#WORKDIR /app
#COPY  ./output/gm/gmserver ./output/gm/gmserver
#COPY  ./output/gm/config.yaml ./output/gm/config.yaml
#USER root:root
#ENTRYPOINT ["./output/gm/gmserver"]


#FROM mcr.microsoft.com/dotnet/core/aspnet:3.1 as battleserver
#WORKDIR /app
#COPY  ./output/bin/linux/battleserver ./output/bin/linux/battleserver
#COPY  ./output/battle ./output/battle
#USER root:root
#ENTRYPOINT ["./output/bin/linux/battleserver"]


# 适用于所有服务的通用容器镜像
FROM mcr.microsoft.com/dotnet/runtime-deps:6.0 as aniwarserver

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list && \
    apt-get update && \
    apt-get install -y procps iproute2 ncat curl && \
    apt-get autoremove && \
    apt-get autoclean

CMD ["bash"]
