#!/usr/bin/env bash
cd $(dirname "$0")
cd ..
#pwd
if [[ $(docker image ls supraworker-base-local -q|wc -l) -lt 1  ]];then
    docker build -t supraworker-base-local:latest -f - . < docker-image/base-image-local/Dockerfile
fi
[ $? -eq 0 ] && \
docker build -t supraworker-local:latest --no-cache	 -f - . <  docker-image/supraworker-image-local/Dockerfile
