Version=v1.24.16

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

all: build

compress:
	upx --best --ultra-brute -o kube-proxy ${DOCKERIZED_OUTPUT_PATH}/kube-proxy

build: compress
	docker build -t 172.22.50.227/system_containers/kube-proxy:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
