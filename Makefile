VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`
BRANCH=`git branch | sed -n '/\* /s///p'`
IMAGE_CONFIG=`cat zke_image.yml`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD} -X github.com/gsmlg-opt/GaoCloud/pkg/zke.gaoCloudVersion=${VERSION} -X 'github.com/gsmlg-opt/GaoCloud/zke/types.imageConfig=${IMAGE_CONFIG}'"
GOSRC = $(shell find . -type f -name '*.go')

build: gaocloud

gaocloud: $(GOSRC) 
	CGO_ENABLED=0 GOOS=linux go build ${LDFLAGS} cmd/gaocloud/gaocloud.go

docker: build-image
	docker push gsmlg-opt/GaoCloud:${BRANCH}

build-image:
	docker build -t gsmlg-opt/GaoCloud:${BRANCH} --build-arg version=${VERSION} --build-arg buildtime=${BUILD} --build-arg goproxy=${GOPROXY} --no-cache .
	docker image prune -f

clean:
	rm -rf gaocloud

clean-image:
	docker rmi gsmlg-opt/GaoCloud:${VERSION}

.PHONY: clean install
