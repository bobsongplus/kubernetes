#!/bin/bash
REGISTRY_SERVER="index.tenxcloud.com"
REGISTRY_USER="system_containers"
K8S_VERSION="v1.18.0"
ETCD_VERSION="3.4.3-0"
CALICO_VERSION="v3.13.2"
HA_BINDPORT="16443"
MASTER_BINDPORT="6443"
KUBE_PROXY_MODE="ipvs"
_ARCH="amd64"
kubeadm_dir="."
tmp_dir="/tmp"
kubeadm_config_file="${kubeadm_dir}/kubeadm-config.yaml"
tmp_file="${tmp_dir}/kubeadm-config.yaml"
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
    cp /kubeadm  /tmp/  1>/dev/null 2>&1
    /tmp/kubeadm reset -f
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
  docker pull \${1}/\${2}/kube-scheduler-${_ARCH}:${K8S_VERSION}
  docker pull \${1}/\${2}/kube-controller-manager-${_ARCH}:${K8S_VERSION}
  docker pull \${1}/\${2}/kube-apiserver-${_ARCH}:${K8S_VERSION}
  docker pull \${1}/\${2}/etcd-${_ARCH}:${ETCD_VERSION}
  docker pull \${1}/\${2}/kubectl-${_ARCH}:${K8S_VERSION}
  docker pull \${1}/\${2}/ctl-${_ARCH}:${CALICO_VERSION}
  docker pull \${1}/\${2}/node-${_ARCH}:${CALICO_VERSION}
  docker pull \${1}/\${2}/cni-${_ARCH}:${CALICO_VERSION}
  if [ \${3} == "master" ]; then
      docker pull  \${1}/\${2}/etcd-${_ARCH}:${ETCD_VERSION}
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
rm /usr/bin/kubeadm 2>/dev/null
rm /tmp/kubeadm 2>/dev/null
echo "Uninstall Node Successfully"
EOF
}

#Deploy kubernetes master
kubeadm_init_configure() {
    local advertiseAddress=""
    local BINDPORT="${SINGLE_MASTER_BINDPORT}"
    if [[ -n "${MASTERS}" ]];then
        BINDPORT="${HA_BINDPORT}"
    fi

    if [[ -n "${ADDRESS}" ]]; then
        advertiseAddress="advertiseAddress: ${ADDRESS}"
    fi

    if [[ -n "${BINDPORT}" ]]; then
        BINDPORT="${BINDPORT}"
    fi
    sed -i -e "s@{{advertiseAddress}}@${advertiseAddress}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{BINDPORT}}@${BINDPORT}@g" "${kubeadm_config_file}"
}

kubeadm_configure() {
    local certSANs=""
    local controlPlaneEndpoint=""
    local serverCertSANs=""
    local apiServerUrl=""
    local apiServerCredential=""
    local clusterName=""
    local networkMode=""
    local networkPlugin="calico"
    local podSubnet=""
    local serviceSubnet=""
    local dnsDomain=""

    if [[ -n "${ARCH}" ]];then
        _ARCH="${ARCH}"
    fi

    if [[ -n "${CERT_EXTRA_SANS}" ]]; then
        serverCertSANs+="serverCertSANs: ["
        certSANs+="certSANs: ["
        delim=""
        sans=${CERT_EXTRA_SANS//,/ }
        for san in ${sans}; do
            tmp="${delim}\"${san}"\"
            certSANs+="${tmp}"
            serverCertSANs+="${tmp}"
            delim=","
        done
    fi
    if [[ -n "${certSANs}" ]];then
        certSANs+=']'
    fi

    if [[ -n "${VIP}" ]]; then
        controlPlaneEndpoint='controlPlaneEndpoint: '"${VIP}:${MASTER_BINDPORT}"
        if [[ -n "${serverCertSANs}" ]]; then
            serverCertSANs+=",\"${VIP}"\"
        else
            serverCertSANs+="serverCertSANs": [\""${VIP}"\"
        fi
    fi

    if [[ -n "${serverCertSANs}" ]]; then
        serverCertSANs+=']'
    fi

    if [[ -n "${SERVER_URL}" ]] && [[ -n "${CREDENTIAL}" ]]; then
        apiServerUrl+="apiServerUrl: ${SERVER_URL}"
        apiServerCredential+="apiServerCredential: ${CREDENTIAL}"
    fi
    if [[ -n "${CLUSTERID}" ]]; then
        clusterName+="clusterName": "${CLUSTERID}"
    fi

    if [[ -n "${NETWORK_MODE}" ]]; then
        networkMode="Mode: ${NETWORK_MODE}"
    fi

    if [[ -n "${POD_CIDR}" ]]; then
        podSubnet+="podSubnet: ${POD_CIDR}"
    fi
    if [[ -n "${SERVICE_CIDR}" ]]; then
        serviceSubnet+="serviceSubnet: ${SERVICE_CIDR}"
    fi
    if [[ -n "${SERVICE_DNS_DOMAIN}" ]]; then
        dnsDomain+="dnsDomain: ${SERVICE_DNS_DOMAIN}"
    fi

    if [[ -n "${NETWORK_PLUGIN}" ]]; then
        networkPlugin="${NETWORK_PLUGIN}"
    fi
    if [[ -n "${MASTERS}" ]];then
        masters+="Masters: ["
        delim=""
        sans=${MASTERS//,/ }
        for san in ${sans}; do
            tmp="${delim}\"${san}"\"
            masters+="${tmp}"
            delim=","
        done
        masters+="]"
    fi
    if [[ -n "${LOADBALANCES}" ]];then
        loadbalances+="LoadBalances: ["
        delim=""
        sans=${LOADBALANCES//,/ }
        for san in ${sans}; do
            tmp="${delim}\"${san}"\"
            loadbalances+="${tmp}"
            delim=","
        done
        loadbalances+="]"
    fi

    if [[ -n "${PROXY_MODE}" ]];then
        KUBE_PROXY_MODE="${PROXY_MODE}"
    fi


    sed -i -e "s@{{K8S_VERSION}}@${K8S_VERSION}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{ARCH}}@${_ARCH}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{REGISTRY_SERVER}}@${REGISTRY_SERVER}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{REGISTRY_USER}}@${REGISTRY_USER}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{certSANs}}@${certSANs}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{controlPlaneEndpoint}}@${controlPlaneEndpoint}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{serverCertSANs}}@${serverCertSANs}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{apiServerUrl}}@${apiServerUrl}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{apiServerCredential}}@${apiServerCredential}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{clusterName}}@${clusterName}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{networkMode}}@${networkMode}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{podSubnet}}@${podSubnet}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{serviceSubnet}}@${serviceSubnet}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{dnsDomain}}@${dnsDomain}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{networkPlugin}}@${networkPlugin}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{masters}}@${masters}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{loadBalances}}@${loadbalances}@g" "${kubeadm_config_file}"
    sed -i -e "s@{{kubeProxMode}}@${KUBE_PROXY_MODE}@g" "${kubeadm_config_file}"
}

kubeadm_init() {
    cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean
/usr/bin/kubeadm init  ${KUBEADM_ARGS} --config "${tmp_file}"
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
mkdir -p $HOME/.kube
if [[ -f $HOME/.kube/config ]];then rm -rf $HOME/.kube/config;fi
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
EOF
    # exit code of kubeadm
    exit 0
    #Normal master mode end
}

InstallBinary() {
    cp /kubeadm  /tmp/  > /dev/null 2>&1
    cp "${kubeadm_config_file}" "${tmp_file}" >/dev/null 2>&1
cat <<EOF
    ${PullImage}
    PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "master"

    if [[ \$? -ne 0  ]];then
        echo "Kubernetes Enterprise Edition cluster deployed  failed!"
        exit 1
    fi
    cp /tmp/kubeadm /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-${_ARCH}:${K8S_VERSION} /usr/bin/kubectl /tmp
    rm -rf $(which kubectl)
    mv /tmp/kubectl /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/etcd-${_ARCH}:${ETCD_VERSION}  /usr/local/bin/etcdctl /tmp
    rm -rf $(which etcdctl)
    mv /tmp/etcdctl /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/ctl-${_ARCH}:${CALICO_VERSION} /calicoctl /tmp
    rm -rf $(which calicoctl)
    mv /tmp/calicoctl /usr/bin/  >/dev/null
    $(CalicoConfig)
    CalicoConfig
EOF
}

Master(){
    kubeadm_init_configure
    kubeadm_configure
    InstallBinary
    kubeadm_init
}


Node(){
    # installbinay simplify
    # InstallBinary
    Join
}

Join() {
  #copy kubeadm from containers to /tmp
  cp /kubeadm  /tmp/ > /dev/null 2>&1
  controlPlaneEndpoint=${MASTER}:${MASTER_BINDPORT}
  if [[ -n "${BINDPORT}" ]]; then
    controlPlaneEndpoint=${MASTER}:${BINDPORT}
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
  else
      apiServerBindPort="--apiserver-bind-port ${HA_BINDPORT}"
  fi



  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "node"
mv /tmp/kubeadm /usr/bin/ > /dev/null 2>&1
/usr/bin/kubeadm join ${controlPlaneEndpoint}  ${apiServerAdvertiseAddress}  ${apiServerBindPort}   --token ${K8S_TOKEN}  --discovery-token-ca-cert-hash ${CA_CERT_HASH}  --control-plane --certificate-key areyoukidingme ${KUBEADM_ARGS}
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
mkdir -p $HOME/.kube
if [[ -f $HOME/.kube/config ]];then rm -rf $HOME/.kube/config;fi
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
EOF
   exit 0
    #join control plane end
  fi

  #Normal worker node
  cat <<EOF
#!/bin/bash;
$(welcome)
welcome
${Clean}
Clean
if [[ "${CONTROLPLANE}" = "true" ]]; then
${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "node"
fi
mv /tmp/kubeadm /usr/bin/ > /dev/null 2>&1
/usr/bin/kubeadm join ${controlPlaneEndpoint} --token ${K8S_TOKEN}  --discovery-token-ca-cert-hash ${CA_CERT_HASH} ${KUBEADM_ARGS}
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
