FROM golang:1.13.7-alpine3.11 AS build

ARG version
ARG buildtime
ARG goproxy

ENV GOPROXY=$goproxy
RUN mkdir -p /go/src/github.com/gsmlg-opt/gaocloud
COPY . /go/src/github.com/gsmlg-opt/gaocloud
WORKDIR /go/src/github.com/gsmlg-opt/gaocloud

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.version=$version -X main.build=$buildtime -X github.com/gsmlg-opt/gaocloud/pkg/zke.singleCloudVersion=$version -X 'github.com/zdnscloud/zke/types.imageConfig=`cat zke_image.yml`'" cmd/singlecloud/singlecloud.go


FROM scratch
COPY --from=build /go/src/github.com/gsmlg-opt/gaocloud/singlecloud /
ENTRYPOINT ["/singlecloud"]
