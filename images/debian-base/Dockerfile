FROM debian:buster-slim

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

ENV DEBIAN_FRONTEND=noninteractive

# Smaller package install size.
COPY excludes /etc/dpkg/dpkg.cfg.d/excludes

RUN echo "deb http://mirrors.aliyun.com/debian/ buster main non-free contrib" > /etc/apt/sources.list
RUN apt-get update -y && apt-get -yy -q install \
    iptables \
    ipset    \
    kmod     \
    ebtables \
    ethtool \
    ca-certificates \
    conntrack \
    e2fsprogs \
    util-linux \
    socat \
    jq \
    ceph-common\
    nfs-common \
    glusterfs-client\
    openssh-client \
    cifs-utils \
    && apt-get upgrade -y && apt-get autoremove -y

# Hold required packages to avoid breaking the installation of packages
RUN apt-mark hold apt gnupg adduser passwd libsemanage1 libcap2

# Cleanup cached and unnecessary files.
RUN apt-get autoremove -y && \
    apt-get clean -y && \
    tar -czf /usr/share/copyrights.tar.gz /usr/share/common-licenses /usr/share/doc/*/copyright && \
    rm -rf \
        /usr/share/doc \
        /usr/share/man \
        /usr/share/info \
        /usr/share/locale \
        /var/lib/apt/lists/* \
        /var/log/* \
        /var/cache/debconf/* \
        /usr/share/common-licenses* \
        /usr/share/bash-completion \
        ~/.bashrc \
        ~/.profile \
        /etc/systemd \
        /lib/lsb \
        /lib/udev \
        /usr/lib/x86_64-linux-gnu/gconv/IBM* \
        /usr/lib/x86_64-linux-gnu/gconv/EBC* && \
    mkdir -p /usr/share/man/man1 /usr/share/man/man2 \
        /usr/share/man/man3 /usr/share/man/man4 \
        /usr/share/man/man5 /usr/share/man/man6 \
        /usr/share/man/man7 /usr/share/man/man8
