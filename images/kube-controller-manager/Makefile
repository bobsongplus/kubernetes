Version=v1.22.1

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

all: build

compress:
	upx --best --ultra-brute -o kube-controller-manager ${DOCKERIZED_OUTPUT_PATH}/kube-controller-manager

build: compress
	docker build -t 172.22.50.227/system_containers/kube-controller-manager-amd64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
