FROM registry.access.redhat.com/ubi8/go-toolset:1.21.11-8.1724662611 as builder

ARG GIT_BRANCH=main

RUN git clone -b ${GIT_BRANCH} --depth 1 https://github.com/ligangty/charon-go.git
RUN cd charon-go && make build 

FROM registry.access.redhat.com/ubi8-minimal:8.10-1052

ARG USER=charon
ARG UID=10000
ARG HOME_DIR=/home/${USER}

WORKDIR ${HOME_DIR}

USER root

RUN microdnf install -y shadow-utils && microdnf clean all
RUN useradd -d ${HOME_DIR} -u ${UID} -g 0 -m -s /bin/bash ${USER} \
    && chown ${USER}:0 ${HOME_DIR} \
    && chmod -R g+rwx ${HOME_DIR} \
    && chmod g+rw /etc/passwd

COPY --from=builder /opt/app-root/src/charon-go/build/charon /usr/local/bin/charon

RUN chmod +x /usr/local/bin/charon

RUN microdnf remove shadow-utils && microdnf clean all

USER ${USER}

ENV HOME=${HOME_DIR} \
    LANG=en_US.UTF-8

CMD ["/usr/local/bin/charon"]