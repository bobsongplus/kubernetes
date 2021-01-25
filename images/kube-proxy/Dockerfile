FROM 192.168.1.220/build-image/debian-iptables-amd64:v12.1.2

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

COPY kube-proxy /usr/local/bin/kube-proxy
# remove iptables-wrapper
RUN echo "deb http://mirrors.aliyun.com/debian/ buster main non-free contrib" > /etc/apt/sources.list \
    && rm -f /usr/sbin/iptables && clean-install iptables
