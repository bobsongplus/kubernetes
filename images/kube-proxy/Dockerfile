FROM registry.k8s.io/build-image/debian-iptables:bullseye-v1.3.0

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

COPY kube-proxy /usr/local/bin/kube-proxy
# remove iptables-wrapper
RUN echo "deb http://mirrors.aliyun.com/debian/ bullseye main non-free contrib" > /etc/apt/sources.list \
    && rm -f /usr/sbin/iptables && clean-install iptables
