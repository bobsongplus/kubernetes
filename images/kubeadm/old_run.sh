#!/bin/bash
REGISTRY_SERVER="index.tenxcloud.com"
# REGISTRY_USER="system_containers"
REGISTRY_USER="k8s.gcr.io"
K8S_VERSION="v1.18.0"
ETCD_VERSION="3.4.3-0"
CALICO_VERSION="v3.13.2"
DEFAULT_BINDPORT="6443"
welcome() {
message="$(cat <<EOF
*******************************************************
||                                                   ||
||                                                   ||
||        Kubernetes Enterprise Edition              ||
||                                                   ||
||                                                   ||
***********************************Kubernetes Enterprise
Welcome to Kubernetes Enterprise Edition Deployment Engine
EOF
)"
echo "welcome() {"
echo "  cat <<\"EOF\""
echo "$message"
echo "EOF"
echo "}"
}

Usage() {
cat <<EOF
Kubernetes Enterprise Edition Deployment Engine\n
\n
Command: \n
    [Option] Join <Master> \n
    [Option] Init [TCEAddress] \n
    [Option] Uninstall \n
\n
Option:\n
    --registry         \t\t Registry server, default is index.tenxcloud.com \n
    --address          \t\t Advertised address of the current machine, if not set, it will get a one automatically\n
    --port             \t\t kube-apiserver port, if not set, it will use default 6443\n
    --version          \t\t kubernetes version that will be deployed\n
    --token            \t\t kubernetes token \n
    --clusterId        \t\t kubernetes cluster name\n
    --control-plane    \t Indicates whether control plane or not \n
    --credential       \t\t credential to access tce api server
EOF
}



Clean=$(cat <<EOF
  Clean() {
    #cp /kubeadm  /tmp/  1>/dev/null 2>&1
    /usr/bin/kubeadm reset -f
  }
EOF
)
eval "${Clean}"

CalicoConfig() {
config="$(cat <<EOF
apiVersion: projectcalico.org/v3
kind: CalicoAPIConfig
metadata:
spec:
  etcdEndpoints: https://127.0.0.1:2379
  etcdCACertFile: /etc/kubernetes/pki/etcd/ca.crt
  etcdCertFile: /etc/kubernetes/pki/etcd/client.crt
  etcdKeyFile:  /etc/kubernetes/pki/etcd/client.key
EOF
)"
echo "CalicoConfig() {"
echo "  mkdir -p /etc/calico"
echo "  cat  > /etc/calico/calicoctl.cfg <<\"EOF\""
echo "$config"
echo "EOF"
echo "}"
}



PullImage=$(cat <<EOF
  PullImage() {
  echo "Pulling Necessary Images from \${1}"
  docker pull \${1}/\${2}/kube-scheduler-amd64:${K8S_VERSION}
  docker pull \${1}/\${2}/kube-controller-manager-amd64:${K8S_VERSION}
  docker pull \${1}/\${2}/kube-apiserver-amd64:${K8S_VERSION}
  docker pull \${1}/\${2}/etcd-amd64:${ETCD_VERSION}
  docker pull \${1}/\${2}/kubectl-amd64:${K8S_VERSION}
  docker pull \${1}/\${2}/ctl-amd64:${CALICO_VERSION}
  docker pull \${1}/\${2}/node-amd64:${CALICO_VERSION}
  docker pull \${1}/\${2}/cni-amd64:${CALICO_VERSION}
  if [ \${3} == "master" ]; then
      docker pull  \${1}/\${2}/etcd-amd64:${ETCD_VERSION}
  fi
  }
EOF
)


uninstall(){
#copy kubeadm from containers to /tmp
cp /kubeadm  /tmp/  > /dev/null 2>&1

cat <<EOF
#!/bin/bash
${Clean}
Clean
rm /tmp/kubeadm 2>/dev/null
echo "Uninstall Node Successfully"
EOF
}

#Deploy kubernetes master
kubeadm_init_config() {
  #copy kubeadm from containers to /tmp
  cp /kubeadm  /tmp/  > /dev/null 2>&1
  # init init.yaml
  cat > /tmp/init.yaml << EOF
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
bootstrapTokens:
- ttl: 0s
nodeRegistration:
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
EOF

   if [[  -n "${ADDRESS}" ]] || [[  -n "${BINDPORT}" ]]; then
     cat >> /tmp/init.yaml << EOF
localAPIEndpoint:
EOF
   fi
   if [[ -n "${ADDRESS}" ]]; then
     cat >> /tmp/init.yaml << EOF
  advertiseAddress: ${ADDRESS}
EOF
  fi
  if [[ -n "${BINDPORT}" ]]; then
    cat >> /tmp/init.yaml << EOF
  bindPort: ${BINDPORT}
EOF
  fi


  cat >> /tmp/init.yaml << EOF
---
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
kubernetesVersion: ${K8S_VERSION}
imageRepository: ${REGISTRY_SERVER}/${REGISTRY_USER}
EOF

  # apiServerCertSANs sets extra Subject Alternative Names for the API Server signing cert.
  if [[ -n "${CERT_EXTRA_SANS}" ]]; then
    cat >> /tmp/init.yaml << EOF
apiServer:
  timeoutForControlPlane: 5m0s
  certSANs:
EOF
    sans=${CERT_EXTRA_SANS//,/ }
    file=$(mktemp /tmp/servers.XXXXXXXX)
    for san in ${sans}; do
       echo "  - $san" >> ${file}
    done
    cat ${file} >> /tmp/init.yaml
    rm -rf ${file}
  fi


    # controlPlaneEndpoint
    if [[ -n "${VIP}" ]]; then
    cat >> /tmp/init.yaml << EOF
controlPlaneEndpoint: ${VIP}
EOF
    fi
    ## controlPlaneEndpoint


    ## etcd args
      cat >> /tmp/init.yaml << EOF
etcd:
  local:
    extraArgs:
      election-timeout: "5000"
      heartbeat-interval: "500"
      max-request-bytes: "3145728"
      quota-backend-bytes: "8589934592"
EOF
    if [[  -n "${CERT_EXTRA_SANS}" ]] || [[  -n "${VIP}" ]]; then
       cat >> /tmp/init.yaml << EOF
    serverCertSANs:
EOF
       if [[  -n "${CERT_EXTRA_SANS}" ]]; then
          certs=${CERT_EXTRA_SANS//,/ }
          tmpfile=$(mktemp /tmp/servers.XXXXXXXX)
          for san in ${certs}; do
              echo "    - $san" >> ${tmpfile}
          done
          cat ${tmpfile} >> /tmp/init.yaml
          rm -rf ${tmpfile}
       fi

       if [[  -n "${VIP}" ]]; then
       cat >> /tmp/init.yaml << EOF
    - ${VIP}
EOF
       fi
    fi

  # add dns type
    if [[ "${DNS_TYPE}" = "kube-dns" ]]; then
    cat >> /tmp/init.yaml << EOF
dns:
  type: kube-dns;
EOF
    else
    cat >> /tmp/init.yaml << EOF
dns:
  type: CoreDNS
EOF
    fi
  ## add dns type

  # webhook url
  if [[ -n "${SERVER_URL}" ]] && [[ -n "${CREDENTIAL}" ]]; then
    if [[ -n "${CLUSTERID}" ]]; then
       cat >> /tmp/init.yaml << EOF
apiServerUrl: ${SERVER_URL}
apiServerCredential: ${CREDENTIAL}
clusterName: ${CLUSTERID}
EOF
    else
       cat >> /tmp/init.yaml << EOF
apiServerUrl: ${SERVER_URL}
apiServerCredential: ${CREDENTIAL}
EOF
    fi
  fi
  ## webhook url

  # network params
  if [[  -n "${NETWORK_PLUGIN}" ]] || [[  -n "${NETWORK_MODE}" ]] || [[  -n "${POD_CIDR}" ]] || [[ -n "${SERVICE_CIDR}" ]] || [[ -n "${SERVICE_DNS_DOMAIN}" ]]; then
     cat >> /tmp/init.yaml << EOF
networking:
EOF
  fi

  if [[ -n "${NETWORK_PLUGIN}" ]]; then
     cat >> /tmp/init.yaml << EOF
  plugin: ${NETWORK_PLUGIN}
EOF
  fi
  if [[ -n "${NETWORK_MODE}" ]]; then
    cat >> /tmp/init.yaml << EOF
  mode: ${NETWORK_MODE}
EOF
  fi
  if [[ -n "${POD_CIDR}" ]]; then
    cat >> /tmp/init.yaml << EOF
  podSubnet: ${POD_CIDR}
EOF
  fi

  if [[ -n "${SERVICE_CIDR}" ]]; then
    cat >> /tmp/init.yaml << EOF
  serviceSubnet: ${SERVICE_CIDR}
EOF
  fi

  if [[ -n "${SERVICE_DNS_DOMAIN}" ]]; then
    cat >> /tmp/init.yaml << EOF
  dnsDomain: ${SERVICE_DNS_DOMAIN}
EOF
  fi
}

kubeadm_init() {
    cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean
cat /tmp/init.yaml
/usr/bin/kubeadm init  --config /tmp/init.yaml
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
EOF
    # exit code of kubeadm
    exit 0
    #Normal master mode end
}

InstallBinary() {
cat <<EOF
    ${PullImage}
    PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "master"

    if [[ \$? -ne 0  ]];then
        echo "Kubernetes Enterprise Edition cluster deployed  failed!"
        exit 1
    fi
    rm -rf $(which kubeadm)
    mv /tmp/kubeadm /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-amd64:${K8S_VERSION} /usr/bin/kubectl /tmp
    rm -rf $(which kubectl)
    mv /tmp/kubectl /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/etcd-amd64:${ETCD_VERSION}  /usr/local/bin/etcdctl /tmp
    rm -rf $(which etcdctl)
    mv /tmp/etcdctl /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/ctl-amd64:${CALICO_VERSION} /calicoctl /tmp
    rm -rf $(which calicoctl)
    mv /tmp/calicoctl /usr/bin/  >/dev/null
    $(CalicoConfig)
    CalicoConfig
EOF
}

Master(){
    InstallBinary
    kubeadm_init_config
    kubeadm_init
}


Node(){
    # installbinay simplify
    InstallBinary
    Join
}

Join() {
  #copy kubeadm from containers to /tmp
  cp /kubeadm  /tmp/ > /dev/null 2>&1
  local controlPlaneEndpoint=""
  if [[ -n "${MASTER}" ]]; then
    if [[ -z "${BINDPORT}" ]]; then
      controlPlaneEndpoint=${MASTER}:${DEFAULT_BINDPORT}
    else
      controlPlaneEndpoint=${MASTER}:${BINDPORT}
    fi
  fi

  if [[ -z "${K8S_TOKEN}" ]]; then
    cat <<EOF
#!/bin/bash
echo "Please set kubernetes token with parameter --token <tokenstring>"
EOF
    exit 1
  fi

  if [[ -z "${CA_CERT_HASH}" ]]; then
      cat <<EOF
#!/bin/bash
echo "Please set kubernetes ca cert hash with parameter --ca-cert-hash sha256:<hash>"
EOF
    exit 1
  fi

  #join control plane begin
  if [[ "${CONTROLPLANE}" = "true" ]]; then

  local apiServerAdvertiseAddress=""
  if [[ -n "${ADDRESS}" ]]; then
     apiServerAdvertiseAddress="--apiserver-advertise-address ${ADDRESS}"
  fi
  local apiServerBindPort=""
  if [[ -n "${BINDPORT}" ]]; then
     apiServerBindPort="--apiserver-bind-port ${BINDPORT}"
  fi



  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "node"
/usr/bin/kubeadm join ${controlPlaneEndpoint}  ${apiServerAdvertiseAddress}  ${apiServerBindPort}   --token ${K8S_TOKEN}  --discovery-token-ca-cert-hash ${CA_CERT_HASH}  --control-plane --certificate-key areyoukidingme
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
EOF
   exit 0
    #join control plane end
  fi

  #Normal worker node
  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "node"
/usr/bin/kubeadm join ${controlPlaneEndpoint} --token ${K8S_TOKEN}  --discovery-token-ca-cert-hash ${CA_CERT_HASH}
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
EOF
exit 0
}



# if there's no valid parameter, it will show help message
if [[ "$#" -le 0 ]] ; then
  echo -e $(Usage)
  exit 0
fi
#welcome message

#dispatch different parameters
 while(( $# > 0 ))
    do
        case "$1" in
          "--registry" )
              REGISTRY_SERVER="$2"
              shift 2;;
          "--address" )
              ADDRESS="$2"
              shift 2;;
          "--port" )
              BINDPORT="$2"
              shift 2;;
          "--version" )
              K8S_VERSION="$2"
              shift 2 ;;
          "--token" )
              K8S_TOKEN="$2"
              shift 2 ;;
          "--ca-cert-hash" )
              CA_CERT_HASH="$2"
              shift 2 ;;
          "--credential" )
              CREDENTIAL="$2"
              shift 2 ;;
          "--clusterId" )
              CLUSTERID="$2"
              shift 2 ;;
          "--control-plane" )
              CONTROLPLANE="true"
              shift ;;
          "Join" )
              if [[ "$#" -le 1 ]]; then
                echo "Please Enter Master Address and Auth Token"
                exit
              fi
              MASTER="$2"
              Node
              exit 0
              shift 3;;
          "Init" )
              if [[ "$#" -gt 1 ]]; then
                SERVER_URL="$2"
              fi
              Master
              exit 0
              shift 2;;
          "Uninstall" )
              uninstall
              exit 0
              shift 1;;
          "welcome" )
              exit 0
              shift 1;;
            * )
                #echo "Invalid parameter: $1"
                echo -e $(Usage)
                exit 1
        esac
    done # end while
