FROM  192.168.1.52/system_containers/debian-hyperkube-base-amd64:0.12.2
# Copy the shell script
RUN touch /etc/ceph/ceph.keyring /etc/ceph/ceph.conf
COPY kube-controller-manager /usr/local/bin/kube-controller-manager

