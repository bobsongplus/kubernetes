Version=v5.1.0

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

compress:
	upx --best --ultra-brute -o kubeadm ${DOCKERIZED_OUTPUT_PATH}/kubeadm

tde: compress
	docker build  -t 192.168.1.220/system_containers/tde:${Version}  .

arm64:
	docker build  -t 192.168.1.220/system_containers/tde-arm64:${Version}  .


clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
