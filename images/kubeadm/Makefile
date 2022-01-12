Version=v5.3.1

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

compress:
	upx --best --ultra-brute -o kubeadm ${DOCKERIZED_OUTPUT_PATH}/kubeadm

tde: compress
	docker build  -t 172.22.50.227/system_containers/tde:${Version}  .

arm64:
	docker build  -t 172.22.50.227/arm64/tde-arm64:${Version} -f Dockerfile-arm64   .


clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
