REGISTRY ?= 172.22.50.227/system_containers
IMAGE ?= $(REGISTRY)/debian-base
ARCH ?= amd64
TAG ?= v2.0.0

base: build

build:
	docker build -t ${IMAGE}-${ARCH}:${TAG} .


