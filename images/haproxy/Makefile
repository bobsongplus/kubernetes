
all:
	docker build -t 172.22.50.227/system_containers/haproxy:2.2.24 .


arm64:
	docker build --platform=linux/arm64 -t 172.22.50.227/arm64/haproxy:2.2.24 -f Dockerfile-arm64 .

