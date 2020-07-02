FROM alpine:3.10

LABEL vendor="TenxCloud Incorporated" \
      maintainer=weiwei@tenxcloud.com

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update && apk add --no-cache libltdl coreutils curl
ADD kubectl docker /usr/bin/
CMD sleep infinity
