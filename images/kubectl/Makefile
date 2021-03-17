Version=v1.20.4
ARCH?=amd64
OUT_DIR?=_output
DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)



all: build

compress:
	upx --best --ultra-brute -o kubectl ${DOCKERIZED_OUTPUT_PATH}/kubectl

build: compress
	docker build -t 192.168.1.220/system_containers/kubectl-amd64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
