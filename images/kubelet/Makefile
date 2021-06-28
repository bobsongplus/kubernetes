Version=v1.20.8
CNI_VERSION=v0.9.1

ARCH?=amd64
OUT_DIR?=_output
DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)



all: build

compress:
	upx --best -o kubelet ${DOCKERIZED_OUTPUT_PATH}/kubelet

build: compress
	docker build -t 192.168.1.220/system_containers/kubelet-amd64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
