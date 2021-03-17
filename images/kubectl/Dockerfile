FROM alpine:3.12 AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update && apk add --no-cache docker
FROM calico/ctl:v3.18.0 AS ctl
FROM alpine:3.12

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update && apk add --no-cache libltdl coreutils curl
COPY --from=builder /usr/bin/docker /usr/bin/docker
COPY --from=ctl     /calicoctl     /usr/bin/calicoctl
ADD kubectl /usr/bin/
CMD sleep infinity
