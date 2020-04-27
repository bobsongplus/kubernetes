Version =v4.2.0

all: tde
compress:
	docker run --rm -v  $(shell pwd):/src/  -w /src 192.168.1.52/tenx_containers/goupx:latest goupx kubeadm

tde:
	docker build -t 192.168.1.52/system_containers/tde-amd64:${Version}  .

arm64:
	docker build -t 192.168.1.52/system_containers/tde-arm64:${Version}  .


clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
