ARG BASE_IMAGE=python
ARG BASE_TAG=3.8.7

FROM ${BASE_IMAGE}:${BASE_TAG}
MAINTAINER Valeriy Soloviov <weldpua2008@gmail.com>

# Make sure noninteractie debian install is used and language variables set
ENV \
  DEBIAN_FRONTEND=noninteractive \
  LANGUAGE=C.UTF-8 LANG=C.UTF-8 LC_ALL=C.UTF-8 \
  LC_CTYPE=C.UTF-8 LC_MESSAGES=C.UTF-8 \
  APP_HOME=/app \
  APP_SETTINGS=flaskconfig.ProductionConfig \
  PYTHONPATH=/app:/app/app \
  PIP=${PIP:-pip3} \
  PATH=${PATH}:/home/deployer/.local/bin

#########################################################
# fixing permissions:
#########################################################
RUN set -ex \
    && addgroup --gid 5500 --system deployer  \
    && adduser --system --ingroup deployer --uid 5500 deployer \
    && mkdir -p /etc/sudoers.d \
    && echo "deployer ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/deployer \
    && chmod 0440 /etc/sudoers.d/deployer \
    && mkdir /logs && chmod 777 /logs && chown deployer:deployer /logs \
    && mkdir -p /tmp/multiproc-tmp \
    && date > /etc/docker_build_date \
    && echo "${BASE_IMAGE}:${BASE_TAG}" > /etc/base_image.txt
USER deployer
WORKDIR ${APP_HOME}

ADD docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ADD requirements.txt requirements.txt
RUN  $PIP install  -r requirements.txt

ADD . .

EXPOSE 8080 8088 8081 8084

CMD ["/usr/local/bin/docker-entrypoint.sh"]
