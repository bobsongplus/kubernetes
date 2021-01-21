Version=v1.18.15
CNI_VERSION=v0.9.0
CNI_TARBALL=cni-plugins-linux-$(ARCH)-$(CNI_VERSION).tgz

ARCH?=amd64
OUT_DIR?=_output
DOCKERIZED_OUTPUT_PATH=$(shell pwd)/../../$(OUT_DIR)/dockerized/bin/linux/$(ARCH)



all: build

cni-tars/$(CNI_TARBALL):
	mkdir -p cni-tars/
	cd cni-tars/ && curl -sSLO --retry 5 https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/${CNI_TARBALL}

compress:
	upx --best -o kubelet ${DOCKERIZED_OUTPUT_PATH}/kubelet

build: compress
	docker build -t 192.168.1.220/system_containers/kubelet-amd64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
