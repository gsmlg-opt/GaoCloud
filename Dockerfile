FROM golang:1.13.7-alpine3.11 AS build

ARG version
ARG buildtime
ARG goproxy

ENV GOPROXY=$goproxy
RUN mkdir -p /go/src/github.com/gsmlg-opt/GaoCloud
COPY . /go/src/github.com/gsmlg-opt/GaoCloud
WORKDIR /go/src/github.com/gsmlg-opt/GaoCloud

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.version=$version -X main.build=$buildtime -X github.com/gsmlg-opt/GaoCloud/pkg/zke.gaoCloudVersion=$version -X 'github.com/gsmlg-opt/GaoCloud/zke/types.imageConfig=`cat zke_image.yml`'" cmd/gaocloud/gaocloud.go


FROM scratch
COPY --from=build /go/src/github.com/gsmlg-opt/GaoCloud/gaocloud /
ENTRYPOINT ["/gaocloud"]
