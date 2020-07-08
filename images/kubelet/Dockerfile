FROM  192.168.1.52/system_containers/debian-base-amd64:v2.0.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

ADD cni-tars/cni-plugins-linux-amd64-v0.8.6.tgz  /opt/cni/bin/
COPY kubelet /kubelet

