#!/usr/bin/env bash
cd $(dirname "$0")
docker build -t local/supraworker.github .
docker tag local/supraworker.github 679315790258.dkr.ecr.us-east-1.amazonaws.com/github-actions:supraworker
docker push 679315790258.dkr.ecr.us-east-1.amazonaws.com/github-actions:supraworker

