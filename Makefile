VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`
BRANCH=`git branch | sed -n '/\* /s///p'`
IMAGE_CONFIG=`cat zke_image.yml`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD} -X gaocloud/pkg/zke.singleCloudVersion=${VERSION} -X 'zke/types.imageConfig=${IMAGE_CONFIG}'"
GOSRC = $(shell find . -type f -name '*.go')

build: gaocloud

gaocloud: $(GOSRC) 
	CGO_ENABLED=0 GOOS=linux go build ${LDFLAGS} cmd/singlecloud/singlecloud.go

docker: build-image
	docker push zdnscloud/gaocloud:${BRANCH}

build-image:
	docker build -t zdnscloud/gaocloud:${BRANCH} --build-arg version=${VERSION} --build-arg buildtime=${BUILD} --build-arg goproxy=${GOPROXY} --no-cache .
	docker image prune -f

clean:
	rm -rf gaocloud

clean-image:
	docker rmi zdnscloud/gaocloud:${VERSION}

.PHONY: clean install
