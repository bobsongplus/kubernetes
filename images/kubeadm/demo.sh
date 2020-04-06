#!/usr/bin/env bash


#Deploying HA Kubernetes example(vip 192.168.1.100)

echo "Deploying Master on 192.168.1.246"
sudo bash -c "$(docker run --rm -e CERT_EXTRA_SANS=api.k8s.io -e VIP=192.168.1.100 -e DNS_TYPE=kube-dns -v /tmp:/tmp 192.168.1.52/system_containers/tde:v4.1.0 --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init  http://192.168.1.103:48000/api/v2/cluster/new)"

echo "Deploying Master on 192.168.1.247"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.52/system_containers/tde:v4.1.0 --registry 192.168.1.52 --token <token> --ca-cert-hash <hash> --control-plane Join 192.168.1.100 )"

echo "Deploying Master on 192.168.1.248"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.52/system_containers/tde:v4.1.0 --registry 192.168.1.52 --token <token> --ca-cert-hash <hash> --control-plane Join 192.168.1.100 )"


#Deploying Kubernetes example

echo "Deploying Master on 192.168.1.246"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.52/system_containers/tde:v4.1.0 --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init http://192.168.1.103:48000/api/v2/cluster/new)"

echo "Deploying Node  on 192.168.1.247"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.52/system_containers/tde:v4.1.0 --registry 192.168.1.52 --token <token> --ca-cert-hash <hash> Join 192.168.1.246 )"