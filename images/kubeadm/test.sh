cp kubeadm-config.tmpl.yaml kubeadm-config.yaml
export CERT_EXTRA_SANS=192.168.1.21
export VIP=192.168.1.119
export POD_CIDR=172.31.0.0/16
export SERVICE_CIDR=10.96.0.0/12
export NETWORK_MODE=ipv4
export SERVICE_DNS_DOMAIN=cluster.local
export MASTERS=192.168.1.21,192.168.2.31
export LOADBALANCES=192.168.1.21
#export NETWORK_PLUGIN=flannel
export NETWORK_PLUGIN=weavenet
export PROXY_MODE=iptables
#./run.sh --address 192.168.1.52  --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init  http://192.168.1.103:48000/api/v2/cluster/new
#./run.sh --address 192.168.1.52  --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init  http://192.168.1.103:48000/api/v2/cluster/new
./run.sh  --registry 192.168.1.52 --credential admin:51c6735b118f11e9824312e2eae0bd5e  Init  http://192.168.4.175:8000/api/v2/cluster/new
cat kubeadm-config.yaml
cp kubeadm-config.tmpl.yaml kubeadm-config.yaml
