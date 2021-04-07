#cp kubeadm-config.tmpl.yaml kubeadm-config.yaml
export CERT_EXTRA_SANS=192.168.1.21,api.k8s.io
export VIP=192.168.1.119
export POD_CIDR=172.31.0.0/16
export SERVICE_CIDR=10.96.0.0/12
export NETWORK_MODE=ipv4
export SERVICE_DNS_DOMAIN=cluster.local
export NETWORK_PLUGIN=flannel
#export NETWORK_PLUGIN=weavenet
#export PROXY_MODE=iptables
#./run.sh --address 192.168.1.52  --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init  http://192.168.1.103:48000/api/v2/cluster/new
#./run.sh --address 192.168.1.52  --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init  http://192.168.1.103:48000/api/v2/cluster/new
#./run.sh  --registry 192.168.1.52 --credential admin:51c6735b118f11e9824312e2eae0bd5e  Init  http://192.168.4.175:8000/api/v2/cluster/new
#./run.sh  --registry 192.168.1.52  Init
./run.sh  --registry 192.168.1.220 --token f6pnhg.tkrfabq1fgj0obyi  --ca-cert-hash sha256:dfed5cde580c6ea77ce57b9305375e88d29edb923a6271bbe8570b108ed12753   --control-plane Join ${VIP}
#./run.sh  --registry 192.168.1.220 --token f6pnhg.tkrfabq1fgj0obyi  --ca-cert-hash sha256:dfed5cde580c6ea77ce57b9305375e88d29edb923a6271bbe8570b108ed12753                   Join ${VIP}
#cat kubeadm-config.yaml
#cp kubeadm-config.tmpl.yaml kubeadm-config.yaml
