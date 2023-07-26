FROM  registry.k8s.io/build-image/debian-base:bullseye-v1.2.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN echo "deb http://mirrors.aliyun.com/debian/ bullseye main non-free contrib" > /etc/apt/sources.list \
    && clean-install util-linux
ADD cni-plugins-linux-amd64-v1.1.1.tgz  /opt/cni/bin/
COPY kubelet /kubelet

