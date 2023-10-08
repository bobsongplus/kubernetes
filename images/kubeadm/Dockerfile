FROM 172.22.96.158/system_containers/alpine:3.18-bash

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

COPY kubeadm.yaml kubeadm run.sh /
RUN chmod 755 run.sh
VOLUME ["/tmp","/tmp"]

ENTRYPOINT ["/bin/bash", "/run.sh"]
