FROM  k8s.gcr.io/build-image/debian-base:buster-v1.8.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN mkdir -p /etc/ceph && touch /etc/ceph/ceph.keyring /etc/ceph/ceph.conf
COPY kube-controller-manager /usr/local/bin/kube-controller-manager

