Version =v1.18.3

ARCH?=amd64
OUT_DIR?=_output

DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)

all: build

compress:
	upx --best --ultra-brute -o kube-scheduler ${DOCKERIZED_OUTPUT_PATH}/kube-scheduler

build:
	docker build -t 192.168.1.52/system_containers/kube-scheduler-${ARCH}:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
