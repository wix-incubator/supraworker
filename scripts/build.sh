#!/usr/bin/env bash
#
# Build locally
#
# Requirements:
# Docker
# yum install -y docker
# service docker start
# git clone https://github.com/wix/supraworker.git
# cd supraworker
#
cd "$(dirname "$0}")" || exit 1
cd ..
docker run -ti --rm   -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.15.11 go build .