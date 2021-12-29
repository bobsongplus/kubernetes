Version=v1.22.1

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

all: build

compress:
	upx --best --ultra-brute -o kube-scheduler ${DOCKERIZED_OUTPUT_PATH}/kube-scheduler

build: compress
	docker build -t 172.22.50.227/system_containers/kube-scheduler-${ARCH}:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
