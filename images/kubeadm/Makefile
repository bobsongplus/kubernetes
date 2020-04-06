Version =v4.2.0

all: tde

compress:
	docker run --rm -v  $(shell pwd):/src/  -w /src 192.168.1.52/tenx_containers/goupx:latest goupx kubeadm

tde:
	docker build -t 192.168.1.52/k8s.gcr.io/tde:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
