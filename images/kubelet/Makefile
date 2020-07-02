Version =v1.18.3
CNI_VERSION=v0.8.6
CNI_TARBALL=cni-plugins-linux-$(ARCH)-$(CNI_VERSION).tgz

all: build

cni-tars/$(CNI_TARBALL):
	mkdir -p cni-tars/
	cd cni-tars/ && curl -sSLO --retry 5 https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/${CNI_TARBALL}

compress:
	docker run --rm -v  $(shell pwd):/src/  -w /src 192.168.1.52/tenx_containers/goupx:latest goupx kubelet

kubelet:
	docker build -t 192.168.1.52/system_containers/hyperkube-amd64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
