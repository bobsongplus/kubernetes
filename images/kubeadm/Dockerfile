FROM 192.168.1.52/system_containers/alpine:3.10-bash

MAINTAINER  "Joshua Andrew"   weiwei@tenxcloud.com

ADD kubeadm-config.yaml /kubeadm-config.yaml
ADD ./kubeadm /kubeadm

COPY run.sh /
RUN chmod 755 run.sh
VOLUME ["/tmp","/tmp"]

ENTRYPOINT ["/bin/bash", "/run.sh"]
