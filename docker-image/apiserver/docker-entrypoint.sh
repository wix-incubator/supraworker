#!/usr/bin/env bash
export PYTHONUNBUFFERED=TRUE
export PYTHONPATH=${PYTHONPATH:-/app}:/app/app:app:${PWD}
export prometheus_multiproc_dir=${prometheus_multiproc_dir:=/tmp/multiproc-tmp}

Green_font_prefix="\033[32m" && Red_font_prefix="\033[31m" && Green_background_prefix="\033[42;37m" && Red_background_prefix="\033[41;37m" && Font_color_suffix="\033[0m"
Info="${Green_font_prefix}[INFO]${Font_color_suffix}"
Error="${Red_font_prefix}[ERROR]${Font_color_suffix}"
echoerr() { if [[ ${QUIET:-0} -ne 1 ]]; then echo -e "${Error} $@" 1>&2; fi }
echoinfo() { if [[ ${QUIET:-0} -ne 1 ]]; then echo -e "${Info} $@" 1>&2; fi }

[[ -e /etc/docker_build_date ]] && echoinfo "Docker image build date: $(cat /etc/docker_build_date), run as: ${USER}"
#[[ -e ${prometheus_multiproc_dir} ]] && rm -rf ${prometheus_multiproc_dir}
#mkdir -p "${prometheus_multiproc_dir}"
#gunicorn --access-logfile - --error-logfile - --log-level debug  --config python:app.gunicorn_config app.app:app

gunicorn --config python:app.gunicorn_config app.app:app