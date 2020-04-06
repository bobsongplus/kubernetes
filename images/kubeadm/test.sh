cp kubeadm-config.tmpl.yaml kubeadm-config.yaml
export CERT_EXTRA_SANS=192.168.1.21
#export VIP=192.168.1.119
export POD_CIDR=172.31.0.0/16
export SERVICE_CIDR=10.96.0.0/12
export NETWORK_MODE=ipv4
export SERVICE_DNS_DOMAIN=cluster.local
#export NETWORK_PLUGIN=weave
#./run.sh --address 192.168.1.52  --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init  http://192.168.1.103:48000/api/v2/cluster/new
#./run.sh --address 192.168.1.52  --registry 192.168.1.52 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init  http://192.168.1.103:48000/api/v2/cluster/new
./run.sh  --registry 192.168.1.52 Init
cat kubeadm-config.yaml
