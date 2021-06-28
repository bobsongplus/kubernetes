FROM  k8s.gcr.io/build-image/debian-base-amd64:buster-v1.7.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN echo "deb http://mirrors.aliyun.com/debian/ buster main non-free contrib" > /etc/apt/sources.list \
    && clean-install util-linux
ADD cni-plugins-linux-amd64-v0.9.1.tgz  /opt/cni/bin/
COPY kubelet /kubelet

