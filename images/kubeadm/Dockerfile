FROM 192.168.1.220/system_containers/alpine:3.12-bash

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

ADD kubeadm-config.yaml /kubeadm-config.yaml
ADD kubeadm /kubeadm
COPY run.sh /
RUN chmod 755 run.sh
VOLUME ["/tmp","/tmp"]

ENTRYPOINT ["/bin/bash", "/run.sh"]
