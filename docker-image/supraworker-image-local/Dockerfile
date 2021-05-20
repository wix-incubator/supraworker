ARG BASE_IMAGE=supraworker-base-local
ARG BASE_TAG=latest

FROM ${BASE_IMAGE}:${BASE_TAG} AS build-env

WORKDIR /go/src/github.com/weldpua2008/supraworker
RUN apk --no-cache add build-base git mercurial gcc

COPY . .

RUN set -ex \
    && go get -d -v ./...  \
    && go install -v ./...  \
    && go build -o /root/supraworker main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=build-env /root/supraworker .
RUN set -ex \
    && adduser -D --shell /bin/bash hadoop \
    && date |tee  /root/date \
    && apk add curl bash
EXPOSE 8080 8088 8081 8084

CMD ["/root/supraworker","-v", "-t"]
