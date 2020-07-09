Version =v1.18.3

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

all: build

compress:
	upx --best --ultra-brute -o kube-controller-manager ${DOCKERIZED_OUTPUT_PATH}/kube-controller-manager

build: compress
	docker build -t 192.168.1.52/system_containers/kube-controller-manager-amd64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
