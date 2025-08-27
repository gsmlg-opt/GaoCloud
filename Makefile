VERSION=v0.0.1-dev
BUILD=`date +%FT%T%z`
BRANCH=`git branch | sed -n '/\* /s///p'`
IMAGE_CONFIG=`cat zke_image.yml`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD} -X github.com/gsmlg-opt/gaocloud/pkg/zke.singleCloudVersion=${VERSION} -X 'github.com/zdnscloud/zke/types.imageConfig=${IMAGE_CONFIG}'"
GOSRC = $(shell find . -type f -name '*.go')

build: singlecloud

singlecloud: $(GOSRC) 
	CGO_ENABLED=0 GOOS=linux go build ${LDFLAGS} cmd/singlecloud/singlecloud.go

docker: build-image
	docker push gsmlg-opt/gaocloud:${BRANCH}

build-image:
	docker build -t gsmlg-opt/gaocloud:${BRANCH} --build-arg version=${VERSION} --build-arg buildtime=${BUILD} --build-arg goproxy=${GOPROXY} --no-cache .
	docker image prune -f

clean:
	rm -rf singlecloud

clean-image:
	docker rmi gsmlg-opt/gaocloud:${VERSION}

.PHONY: clean install
