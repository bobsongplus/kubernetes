FROM alpine:3.18 AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
  apk update && apk add --no-cache docker

FROM calico/ctl:v3.24.6 AS ctl
FROM ghcr.io/containerd/nerdctl:v1.5.0  AS nerdctl
FROM alpine:3.18

LABEL vendor="TenxCloud Incorporated" \
  maintainer=weiwei@tenxcloud.com

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
  apk update && apk add --no-cache libltdl coreutils curl bind-tools
COPY --from=builder /usr/bin/docker              /usr/bin/docker
COPY --from=ctl     /calicoctl                  /usr/bin/calicoctl
COPY --from=nerdctl /usr/local/bin/nerdctl      /usr/bin/nerdctl
ADD nerdctl.toml /etc/nerdctl/nerdctl.toml 
ADD kubectl /usr/bin/
CMD sleep infinity
