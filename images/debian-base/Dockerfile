FROM debian:jessie

MAINTAINER Joshua Andrew  weiwei@tenxcloud.com

ADD sources.list  /etc/apt/sources.list
RUN apt-get update -y && apt-get install -y wget
RUN wget -q -O- 'http://mirrors.163.com/ceph/keys/release.asc' | apt-key add -
RUN echo deb http://mirrors.163.com/ceph/debian-luminous/ jessie main > /etc/apt/sources.list.d/ceph.list
RUN DEBIAN_FRONTEND=noninteractive apt-get update -y \
    && DEBIAN_FRONTEND=noninteractive apt-get -yy -q install \
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
    && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y \
    && DEBIAN_FRONTEND=noninteractive apt-get autoremove -y \
    && DEBIAN_FRONTEND=noninteractive apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* # CACHEBUST

COPY cni-bin/bin /opt/cni/bin
RUN chmod -R 755 /opt/cni/bin/

# Create symlinks for each hyperkube server
# Also create symlinks to /usr/local/bin/ where the server image binaries live, so the hyperkube image may be
# used instead of gcr.io/google_containers/kube-* without any modifications.
# TODO: replace manual symlink creation with --make-symlink command once
# cross-building with qemu supports go binaries. See #28702
# RUN /hyperkube --make-symlinks
RUN ln -s /hyperkube /apiserver \
 && ln -s /hyperkube /controller-manager \
 && ln -s /hyperkube /aggregator \
 && ln -s /hyperkube /kubectl \
 && ln -s /hyperkube /kubelet \
 && ln -s /hyperkube /proxy \
 && ln -s /hyperkube /scheduler \
 && ln -s /hyperkube /usr/local/bin/kube-apiserver \
 && ln -s /hyperkube /usr/local/bin/kube-controller-manager \
 && ln -s /hyperkube /usr/local/bin/kube-aggregator \
 && ln -s /hyperkube /usr/local/bin/kubectl \
 && ln -s /hyperkube /usr/local/bin/kubelet \
 && ln -s /hyperkube /usr/local/bin/kube-proxy \
 && ln -s /hyperkube /usr/local/bin/kube-scheduler
