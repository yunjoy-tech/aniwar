#BUILDVER=`git rev-parse HEAD`
#BUILDDATE=`date +%Y-%m-%d_%H_%M_%S`
#VERSION=${BUILDVER}@${BUILDDATE}

CGO_ENABLED=0
export VERSION=${version}
export DOCKER_TAG=${image}
export PUBLIC=${public}
export BRANCH=${branch}
export GO_VERSION=1.18.3

# make for res
res:
	bash ./res.sh

# build for win
win_release:
	export GOOS=windows; export GOARCH=amd64; \
	bash ./make.sh win release

win_develop:
	export GOOS=windows; export GOARCH=amd64; \
	bash ./make.sh win develop


# build for linux
linux_develop:
	export GOOS=linux; export GOARCH=amd64; \
	bash ./make.sh linux develop

linux_release:
	export GOOS=linux; export GOARCH=amd64; \
	bash ./make.sh linux release



# build for mac
mac:
	export GOOS=darwin; export GOARCH=amd64; \
	bash ./make.sh mac


lint:
	# binary will be $(go env GOPATH)/bin/golangci-lint
	#curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.47.3
	chmod +x ./golangci-lint \
	./golangci-lint --version \
	./golangci-lint run ./src --skip-files ".*_test.go" --skip-dirs "tools" --skip-dirs "test" --allow-parallel-runners

startwin:
	start.bat

start:
	bash ./start.sh linux

startmac:
	bash ./start.sh mac

stop:
	bash ./stop.sh

image:
	bash ./image.sh

tar:
	bash ./pkg.sh

gmstop:
	killall gmserver

gmstart:
	cd output/gm && chmod +x gmserver && bash ./start.sh


gmbuild:
	mkdir -p -m 777 output/gm
	cd tools/gin-vue-admin &&  bash ./make.sh

version:
	bash ./version.sh

battleCopyFromClient:
	bash ./battleCopyFromClient.sh

logmstart:
	nohup python3 script/logmonitor.py >/dev/null 2>&1 &

logmstop:
	kill -9 `ps aux |grep -i script/logmonitor.py |grep -v grep |awk '{print $2}'`
mod:
	cd ../musae
	git pull
	go mod tidy
	go mod vendor
proto:
	cd ./src/proto/protocol/
	git pull

