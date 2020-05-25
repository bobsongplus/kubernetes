K8S_VERSION=1.18.3
Version=v5.0

all: tde
compress:
	docker run --rm -v  $(shell pwd):/src/  -w /src 192.168.1.52/tenx_containers/goupx:latest goupx kubeadm

tde:
	docker build --build-arg k8s_version=${K8S_VERSION} -t 192.168.1.52/system_containers/tde:${Version}  .

arm64:
	docker build --build-arg k8s_version=${K8S_VERSION} -t 192.168.1.52/system_containers/tde-arm64:${Version}  .


clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
