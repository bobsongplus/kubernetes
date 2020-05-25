FROM 192.168.1.52/system_containers/alpine:3.10-bash

MAINTAINER  "Joshua Andrew"   weiwei@tenxcloud.com

ADD kubeadm-config.yaml /kubeadm-config.yaml
ADD ./kubeadm /kubeadm
ARG k8s_version
ENV K8S_VERSION_VAR=$k8s_version
COPY run.sh /
RUN chmod 755 run.sh
VOLUME ["/tmp","/tmp"]

ENTRYPOINT ["/bin/bash", "/run.sh"]
