FROM golang:1.15-alpine AS build-env

WORKDIR /go/src/github.com/wix/supraworker

COPY . .

RUN apk --no-cache add build-base git mercurial gcc

# docker image build --no-cache --tag supraworker:snapshot --build-arg GIT_COMMIT=$(git log -1 --format=%H) .
ARG GIT_COMMIT

RUN go get -d -v ./... && \
    go install -v ./... && \
    export GIT_COMMIT_LOACAL=$(git log -1 --format=%H) && \
    export GIT_COMMIT=${GIT_COMMIT:-$GIT_COMMIT_LOACAL} && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /root/supraworker -ldflags="-X github.com/wix/supraworker/cmd.GitCommit=${GIT_COMMIT}" main.go

FROM alpine:latest

# Bring in metadata via --build-arg
ARG BRANCH=unknown
ARG IMAGE_CREATED=unknown
ARG IMAGE_REVISION=unknown
ARG IMAGE_VERSION=unknown

# Configure image labels
LABEL \
    # https://github.com/opencontainers/image-spec/blob/master/annotations.md
    branch=$branch \
    maintainer="Valeriy Soloviov" \
    org.opencontainers.image.authors="Valeriy Soloviov" \
    org.opencontainers.image.created=$IMAGE_CREATED \
    org.opencontainers.image.description="The abstraction layer around jobs, allows pull a job from your API periodically, call-back your API, observe execution time and to control concurrent execution." \
    org.opencontainers.image.documentation="https://github.com/wix/supraworker/" \
    org.opencontainers.image.licenses="Apache License 2.0" \
    org.opencontainers.image.revision=$IMAGE_REVISION \
    org.opencontainers.image.source="https://github.com/wix/supraworker/" \
    org.opencontainers.image.title="Supraworker" \
    org.opencontainers.image.url="https://github.com/wix/supraworker/" \
    org.opencontainers.image.vendor="Supraworker" \
    org.opencontainers.image.version=$IMAGE_VERSION

# Default image environment variable settings
ENV org.opencontainers.image.created=$IMAGE_CREATED \
    org.opencontainers.image.revision=$IMAGE_REVISION \
    org.opencontainers.image.version=$IMAGE_VERSION


WORKDIR /root/

# Copy source
COPY --from=build-env /root/supraworker .

RUN adduser -D --shell /bin/bash hadoop && \
    apk --no-cache add curl bash

# Set entrypoint
ENTRYPOINT ["/root/supraworker"]
