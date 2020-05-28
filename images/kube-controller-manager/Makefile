Version =v1.18.3

all: build

compress:
	docker run --rm -v  $(shell pwd):/src/  -w /src 192.168.1.52/tenx_containers/goupx:latest goupx kube-controller-manager
build:
	docker build -t 192.168.1.52/system_containers/kube-controller-manager-amd64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
