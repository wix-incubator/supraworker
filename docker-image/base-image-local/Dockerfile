FROM golang:1.14-alpine

WORKDIR /go/src/github.com/weldpua2008/supraworker

RUN apk --no-cache add build-base git mercurial gcc
COPY . .
RUN go get -d -v ./... && \
    go install -v ./... && \
    go build -o /root/supraworker main.go && \
    rm -f /root/supraworker && \
    rm -rf /go/src/github.com/weldpua2008/supraworker && \
    mkdir -p /go/src/github.com/weldpua2008/supraworker
