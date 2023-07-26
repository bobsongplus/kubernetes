FROM  registry.k8s.io/build-image/debian-base:bullseye-v1.2.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN mkdir -p /etc/ceph && touch /etc/ceph/ceph.keyring /etc/ceph/ceph.conf
COPY kube-controller-manager /usr/local/bin/kube-controller-manager

