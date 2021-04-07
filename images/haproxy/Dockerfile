FROM haproxy:2.2.11-alpine

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

COPY runit/ /
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache runit && rm -rf /var/cache/apk/* &&  \
    wget -qc -O /usr/bin/confd https://github.com/kelseyhightower/confd/releases/download/v0.16.0/confd-0.16.0-linux-amd64 && \
    chmod +x /usr/bin/confd

ENTRYPOINT ["start_runit"]