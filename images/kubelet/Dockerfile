FROM  192.168.1.52/build-image/debian-base-amd64:v2.1.3

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN echo "deb http://mirrors.aliyun.com/debian/ buster main non-free contrib" > /etc/apt/sources.list \
    && clean-install util-linux
ADD cni-tars/cni-plugins-linux-amd64-v0.8.6.tgz  /opt/cni/bin/
COPY kubelet /kubelet

