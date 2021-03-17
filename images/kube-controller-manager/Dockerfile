FROM  192.168.1.220/build-image/debian-base-amd64:buster-v1.2.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN mkdir -p /etc/ceph && touch /etc/ceph/ceph.keyring /etc/ceph/ceph.conf
COPY kube-controller-manager /usr/local/bin/kube-controller-manager

