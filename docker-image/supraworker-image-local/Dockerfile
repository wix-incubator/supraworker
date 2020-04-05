FROM supraworker-base-local:latest AS build-env

WORKDIR /go/src/github.com/weldpua2008/supraworker

COPY . .

RUN go get -d -v ./... && \
    go install -v ./... && \
    go build -o /root/supraworker main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=build-env /root/supraworker .
RUN adduser -D --shell /bin/bash hadoop && \
    date |tee  /root/date
#&& \
#    apk add curl bash
CMD ["/root/supraworker","-v", "-t"]
