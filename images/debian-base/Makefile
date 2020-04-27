REGISTRY?=192.168.1.52/system_containers
IMAGE?=$(REGISTRY)/debian-hyperkube-base
ARCH?=amd64
TAG=0.12.2
CNI_TARBALL=cni-plugins-linux-$(ARCH)-$(CNI_VERSION).tgz
CNI_VERSION=v0.8.5
TEMP_DIR:=$(shell mktemp -d)
CNI_TARBALL=cni-plugins-linux-$(ARCH)-$(CNI_VERSION).tgz


cni-tars/$(CNI_TARBALL):
	mkdir -p cni-tars/
	cd cni-tars/ && curl -sSLO --retry 5 https://storage.googleapis.com/k8s-artifacts-cni/release/${CNI_VERSION}/${CNI_TARBALL}

build: cni-tars/$(CNI_TARBALL)
	cp Dockerfile $(TEMP_DIR)
	cp sources.list $(TEMP_DIR)
	cd $(TEMP_DIR) && sed -i "s|BASEIMAGE|$(BASEIMAGE)|g" Dockerfile

ifeq ($(CACHEBUST),1)
	cd ${TEMP_DIR} && sed -i.back "s|CACHEBUST|$(shell uuidgen)|g" Dockerfile
endif
	mkdir -p ${TEMP_DIR}/cni-bin/bin
	tar -xz -C ${TEMP_DIR}/cni-bin/bin -f "cni-tars/${CNI_TARBALL}"

ifneq ($(ARCH),amd64)
	# Register /usr/bin/qemu-ARCH-static as the handler for non-x86 binaries in the kernel
	$(SUDO) ../../third_party/multiarch/qemu-user-static/register/register.sh --reset
endif
	docker build -t $(IMAGE)-$(ARCH):$(TAG) $(TEMP_DIR)
	rm -rf $(TEMP_DIR)
