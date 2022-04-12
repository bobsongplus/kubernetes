FROM  k8s.gcr.io/build-image/debian-base:buster-v1.9.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN echo "deb http://mirrors.aliyun.com/debian/ buster main non-free contrib" > /etc/apt/sources.list \
    && clean-install util-linux
ADD cni-plugins-linux-amd64-v1.1.1.tgz  /opt/cni/bin/
COPY kubelet /kubelet

