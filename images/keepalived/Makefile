all:
	docker build -t 172.22.50.227/system_containers/keepalived:2.2.8 .


arm64:
	docker build --platform=linux/arm64 -t 172.22.50.227/arm64/keepalived:2.2.8 -f Dockerfile-arm64  .

