FROM 192.168.1.220/build-image/debian-iptables-amd64:v12.1.2

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

COPY kube-proxy /usr/local/bin/kube-proxy
# remove iptables-wrapper
RUN rm -f /usr/sbin/iptables && clean-install iptables
