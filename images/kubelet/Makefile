Version=v1.20.8
CNI_VERSION=v0.9.1

ARCH?=amd64
OUT_DIR?=_output
DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)



all: build

compress:
	upx --best -o kubelet ${DOCKERIZED_OUTPUT_PATH}/kubelet

build: compress
	docker build -t 172.22.50.227/system_containers/kubelet-amd64:${Version}  .

arm64:
	docker build  --platform=linux/arm64 -t 172.22.50.227/arm64/kubelet:${Version} -f Dockerfile-arm64 .

mips64le:
	docker build --platform=linux/mips64le -t 172.22.50.227/mips64le/kubelet:${Version} -f Dockerfile-mips64le .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
