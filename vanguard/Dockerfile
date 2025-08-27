#use golang to build, then copy to alpine
#build in ubuntu will cause lib64 dependency
FROM golang:alpine AS build

RUN mkdir -p /go/src/github.com/zdnscloud/vanguard
COPY . /go/src/github.com/zdnscloud/vanguard

WORKDIR /go/src/github.com/zdnscloud/vanguard
RUN CGO_ENABLED=0 GOOS=linux go build cmd/vanguard/vanguard.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=build /go/src/github.com/zdnscloud/vanguard/vanguard /usr/local/bin/

EXPOSE 53/udp
EXPOSE 9000/tcp
ENTRYPOINT ["vanguard"]
