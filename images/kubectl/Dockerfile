FROM alpine:3.18 AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
  apk update && apk add --no-cache docker

FROM calico/ctl:v3.23.2 AS ctl
FROM alpine:3.18

LABEL vendor="TenxCloud Incorporated" \
  maintainer=weiwei@tenxcloud.com

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
  apk update && apk add --no-cache libltdl coreutils curl bind-tools
COPY --from=builder /usr/bin/docker /usr/bin/docker
COPY --from=ctl     /calicoctl     /usr/bin/calicoctl
ADD nerdctl.toml /etc/nerdctl/nerdctl.toml 
ADD kubectl /usr/bin/
ADD nerdctl /usr/bin/
CMD sleep infinity
