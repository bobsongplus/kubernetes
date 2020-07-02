FROM  192.168.1.52/system_containers/debian-base-amd64:v2.0.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN touch /etc/ceph/ceph.keyring /etc/ceph/ceph.conf
COPY kube-controller-manager /usr/local/bin/kube-controller-manager

