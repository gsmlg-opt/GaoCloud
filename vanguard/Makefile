VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD}"
GOSRC = $(shell find . -type f -name '*.go' -not -path "./cmd/*" -not -path "./tools/*")

build: vanguard vgclient

vgclient: 
	go build cmd/vgclient/vgclient.go

vanguard: $(GENERATE_FILE) $(GOSRC) cmd/vanguard/vanguard.go
	go build ${LDFLAGS} cmd/vanguard/vanguard.go 

test:
	go test -v -timeout 60s -race ./...

install:
	cd cmd/vanguard && go install 

docker:
	docker build -t zdnscloud/vanguard:v0.2 .
	docker image prune -f
	docker push zdnscloud/vanguard:v0.2

build-image:
	docker build -t zdnscloud/vanguard:v0.2 --build-arg version=${VERSION} --build-arg buildtime=${BUILD} .
	docker image prune -f

clean:
	rm -rf vanguard
	rm -rf vgclient

.PHONY: clean install
