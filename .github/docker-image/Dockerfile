FROM ubuntu:18.04
ENV RUNNER_VERSION=2.280.3
ENV RUNNER_URL=https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz

# Install prerequisites
RUN apt-get update \
    && apt-get install -y \
        apt-transport-https \
        ca-certificates \
        openssh-client \
        curl \
        git \
        software-properties-common

# docker repos
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - \
    && echo "deb [arch=amd64] https://download.docker.com/linux/ubuntu xenial stable" >> /etc/apt/sources.list.d/additional-repositories.list \
    && echo "deb http://ftp-stud.hs-esslingen.de/ubuntu xenial main restricted universe multiverse" >> /etc/apt/sources.list.d/official-package-repositories.list \
    && apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 437D05B5 \
    && apt-get update

# docker
RUN apt-get -y install docker-ce

# Github runner
WORKDIR /app
RUN curl -0 -L -o /tmp/actions-runner.tar.gz $RUNNER_URL \
  && tar xzf /tmp/actions-runner.tar.gz \
  && rm /tmp/actions-runner.tar.gz
RUN bash /app/bin/installdependencies.sh \
    && rm -rf /var/lib/apt/lists/*

# Prepare entrypoint
COPY entrypoint.sh entrypoint.sh
RUN chmod +x entrypoint.sh

# setup user and permissions
ARG uid=1000
ARG gid=1000
ARG dgid=989
RUN groupadd --gid $gid github \
  && useradd --create-home --uid $uid --gid $gid --groups docker github

# prepare workspace
ENV WORK_DIR=/workspace
WORKDIR $WORK_DIR
RUN chown -R github:github /app $WORK_DIR

USER github
ENTRYPOINT ["/app/entrypoint.sh"]
