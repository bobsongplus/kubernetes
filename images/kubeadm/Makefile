Version=v1.24.16

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

compress:
	upx --best --ultra-brute -o kubeadm ${DOCKERIZED_OUTPUT_PATH}/kubeadm

tde: compress
	DOCKER_BUILDKIT=0 docker build  -t 172.22.96.158/system_containers/tde:${Version}  .

arm64:
	docker build  -t 172.22.96.158/arm64/tde:${Version} -f Dockerfile-arm64   .


clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
