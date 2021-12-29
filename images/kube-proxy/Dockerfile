FROM k8s.gcr.io/build-image/debian-iptables:buster-v1.6.5

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

COPY kube-proxy /usr/local/bin/kube-proxy
# remove iptables-wrapper
RUN echo "deb http://mirrors.aliyun.com/debian/ buster main non-free contrib" > /etc/apt/sources.list \
    && rm -f /usr/sbin/iptables && clean-install iptables
