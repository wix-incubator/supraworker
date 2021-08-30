#!/usr/bin/env bash

# Verify repo URL and token have been given, otherwise we must be interactive mode.
if [[ -z "$GITHUB_REPOSITORY" ]];then
    echo "GITHUB_REPOSITORY cannot be empty"
    exit 1
elif [[ -z "$GITHUB_TOKEN" ]]; then
  echo "GITHUB_TOKEN cannot be empty"
  exit 1
fi

if [[ -z "$RUNNER_NAME" ]]; then
	export RUNNER_NAME="$(hostname)"
fi
START_TIME=$(date +%s)

function on_exit(){
    local _exit_code=${1:-1}
    local runtime=$(($(date +%s) - START_TIME))
    echo "++++++++++++++++++++++++++++++++++++++++++++++
Uptime ${runtime} seconds
++++++++++++++++++++++++++++++++++++++++++++++"
    if [[ ! -z "$GITHUB_TOKEN" ]]; then
        /app/config.sh remove --token $GITHUB_TOKEN
    fi
    exit ${_exit_code}
}
trap 'on_exit $?' EXIT HUP TERM INT

printf "Configuring GitHub Runner for $GITHUB_REPOSITORY\n"
printf "\tRunner Name: $RUNNER_NAME\n\tWorking Directory: $WORK_DIR\n"
/app/config.sh  --unattended --replace  \
    --url $GITHUB_REPOSITORY \
    --token $GITHUB_TOKEN \
    --work $WORK_DIR/$RUNNER_NAME \
    --name $RUNNER_NAME --runasservice  --labels

printf "Executing GitHub Runner for $GITHUB_REPOSITORY\n"
/app/run.sh
