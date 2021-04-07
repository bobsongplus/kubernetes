#!/bin/bash
REGISTRY_SERVER="index.tenxcloud.com"
REGISTRY_USER="system_containers"
K8S_VERSION="v1.20.4"
ETCD_VERSION="3.4.13-3"
INTERNAL_BINDPORT="16443"
DEFAULT_BINDPORT="6443"
ARCH="amd64"
kubeadm_config_tmpl="kubeadm-config.yaml"
kubeadm_config_file="/tmp/kubeadm-config.yaml"
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
  etcdCertFile: /etc/kubernetes/pki/etcd/healthcheck-client.crt
  etcdKeyFile:  /etc/kubernetes/pki/etcd/healthcheck-client.key
EOF
)"
echo "CalicoConfig() {"
echo "  mkdir -p /etc/calico"
echo "  cat  > /etc/calico/calicoctl.cfg <<\"EOF\""
echo "$config"
echo "EOF"
echo "}"
}



# ${REGISTRY_SERVER} ${REGISTRY_USER}
PullImage=$(cat <<EOF
  PullImage() {
  echo "Pulling Necessary Images from \${1}"
  docker pull \${1}/\${2}/kube-proxy-${ARCH}:${K8S_VERSION}
  docker pull \${1}/\${2}/kubelet-${ARCH}:${K8S_VERSION}
  docker pull \${1}/\${2}/kubectl-${ARCH}:${K8S_VERSION}
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


init_configure() {
    local advertiseAddress=""
    local bindPort=""
    local certSANs=""
    local controlPlaneEndpoint=""
    local serverCertSANs=""
    local apiServerUrl=""
    local apiServerCredential=""
    local clusterName=""
    local networkMode=""
    local networkPlugin="calico"
    local podSubnet=""
    local podExtraSubnet=""
    local serviceSubnet=""
    local nodeSubnet=""
    local dnsDomain=""
    local kubeProxyMode="ipvs"


    if [[ -n "${ADDRESS}" ]]; then
        advertiseAddress="advertiseAddress: ${ADDRESS}"
    fi
    # none HA
    if [[ -n "${BINDPORT}" ]]; then
        bindPort="${BINDPORT}"
    else
        bindPort=${DEFAULT_BINDPORT}
    fi

    if [[ -n "${CERT_EXTRA_SANS}" ]]; then
        serverCertSANs+="serverCertSANs: ["
        certSANs+="certSANs: ["
        delim=""
        sans=${CERT_EXTRA_SANS//,/ }
        for san in ${sans}; do
            tmp="${delim}${san}"
            certSANs+=${tmp}
            serverCertSANs+=${tmp}
            delim=","
        done
    fi
    if [[ -n "${certSANs}" ]];then
        certSANs+="]"
    fi
    # HA
    if [[ -n "${VIP}" ]]; then
        if [[ -n "${BINDPORT}" ]]; then
           controlPlaneEndpoint="controlPlaneEndpoint: ${VIP}:${BINDPORT}"
        else
           controlPlaneEndpoint="controlPlaneEndpoint: ${VIP}":${DEFAULT_BINDPORT}
        fi
        bindPort=${INTERNAL_BINDPORT}
        if [[ -n "${serverCertSANs}" ]]; then
            serverCertSANs+=",${VIP}"
        else
            serverCertSANs+="serverCertSANs: [${VIP}"
        fi
    fi

    if [[ -n "${serverCertSANs}" ]]; then
        serverCertSANs+="]"
    fi

    if [[ -n "${SERVER_URL}" ]] && [[ -n "${CREDENTIAL}" ]]; then
        apiServerUrl+="apiServerUrl: ${SERVER_URL}"
        apiServerCredential+="apiServerCredential: ${CREDENTIAL}"
    fi
    if [[ -n "${CLUSTERID}" ]]; then
        clusterName+="clusterName: ${CLUSTERID}"
    fi

    if [[ -n "${NETWORK_MODE}" ]]; then
        networkMode="mode: ${NETWORK_MODE}"
    fi

    if [[ -n "${POD_CIDR}" ]]; then
        podSubnet+="podSubnet: ${POD_CIDR}"
    fi
    if [[ -n "${POD_EXTRA_CIDR}" ]]; then
        podExtraSubnet+="podExtraSubnet: ${POD_EXTRA_CIDR}"
    fi
    if [[ -n "${SERVICE_CIDR}" ]]; then
        serviceSubnet+="serviceSubnet: ${SERVICE_CIDR}"
    fi
    if [[ -n "${NODE_CIDR}" ]]; then
        nodeSubnet+="nodeSubnet: ${NODE_CIDR}"
    fi
    if [[ -n "${SERVICE_DNS_DOMAIN}" ]]; then
        dnsDomain+="dnsDomain: ${SERVICE_DNS_DOMAIN}"
    fi

    if [[ -n "${NETWORK_PLUGIN}" ]]; then
        networkPlugin="${NETWORK_PLUGIN}"
    fi
    if [[ -n "${PROXY_MODE}" ]];then
        kubeProxyMode=${PROXY_MODE}
    fi

    cp "${kubeadm_config_tmpl}" "${kubeadm_config_file}" >/dev/null 2>&1
    # mac sed diff linux sed usage
    sed -i  "s@{{advertiseAddress}}@${advertiseAddress}@g" "${kubeadm_config_file}"
    sed -i  "s@{{bindPort}}@${bindPort}@g" "${kubeadm_config_file}"
    sed -i  "s@{{kubernetesVersion}}@${K8S_VERSION}@g" "${kubeadm_config_file}"
    sed -i  "s@{{REGISTRY_SERVER}}@${REGISTRY_SERVER}@g" "${kubeadm_config_file}"
    sed -i  "s@{{REGISTRY_USER}}@${REGISTRY_USER}@g" "${kubeadm_config_file}"
    sed -i  "s@{{certSANs}}@${certSANs}@g" "${kubeadm_config_file}"
    sed -i  "s@{{controlPlaneEndpoint}}@${controlPlaneEndpoint}@g" "${kubeadm_config_file}"
    sed -i  "s@{{serverCertSANs}}@${serverCertSANs}@g" "${kubeadm_config_file}"
    sed -i  "s@{{apiServerUrl}}@${apiServerUrl}@g" "${kubeadm_config_file}"
    sed -i  "s@{{apiServerCredential}}@${apiServerCredential}@g" "${kubeadm_config_file}"
    sed -i  "s@{{clusterName}}@${clusterName}@g" "${kubeadm_config_file}"
    sed -i  "s@{{networkMode}}@${networkMode}@g" "${kubeadm_config_file}"
    sed -i  "s@{{podSubnet}}@${podSubnet}@g" "${kubeadm_config_file}"
    sed -i  "s@{{podExtraSubnet}}@${podExtraSubnet}@g" "${kubeadm_config_file}"
    sed -i  "s@{{serviceSubnet}}@${serviceSubnet}@g" "${kubeadm_config_file}"
    sed -i  "s@{{nodeSubnet}}@${nodeSubnet}@g" "${kubeadm_config_file}"
    sed -i  "s@{{dnsDomain}}@${dnsDomain}@g" "${kubeadm_config_file}"
    sed -i  "s@{{networkPlugin}}@${networkPlugin}@g" "${kubeadm_config_file}"
    sed -i  "s@{{kubeProxyMode}}@${kubeProxyMode}@g" "${kubeadm_config_file}"

}

do_init() {
    cp /kubeadm  /tmp/  > /dev/null 2>&1
    cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean
/tmp/kubeadm config images pull --config "${kubeadm_config_file}"
/tmp/kubeadm init  ${KUBEADM_ARGS} --config "${kubeadm_config_file}"
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
   exit 1
fi
mkdir -p $HOME/.kube
if [[ -f $HOME/.kube/config ]];then rm -f $HOME/.kube/config;fi
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
EOF
}

install_binary() {
    cat <<EOF
    mv /tmp/kubeadm /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-${ARCH}:${K8S_VERSION} /usr/bin/kubectl /tmp/kubectl
    rm -f /usr/bin/kubectl
    mv /tmp/kubectl /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/etcd-${ARCH}:${ETCD_VERSION}  /usr/local/bin/etcdctl /tmp/etcdctl
    rm -f /usr/bin/etcdctl
    mv /tmp/etcdctl /usr/bin/  >/dev/null

    docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-${ARCH}:${K8S_VERSION} /usr/bin/calicoctl /tmp/calicoctl
    rm -f /usr/bin/calicoctl
    mv /tmp/calicoctl /usr/bin/  >/dev/null
    $(CalicoConfig)
    CalicoConfig
EOF
}

Init(){
    init_configure
    do_init
    install_binary
}

Join() {
  #copy kubeadm from containers to /tmp
  cp /kubeadm  /tmp/ > /dev/null 2>&1
  local controlPlaneEndpoint=${MASTER}:${DEFAULT_BINDPORT}
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
      apiServerBindPort="--apiserver-bind-port ${INTERNAL_BINDPORT}"
  fi


  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

/tmp/kubeadm config images pull --image-repository=${REGISTRY_SERVER}/${REGISTRY_USER}
/tmp/kubeadm join ${KUBEADM_ARGS} ${controlPlaneEndpoint}  ${apiServerAdvertiseAddress}  ${apiServerBindPort}   --token ${K8S_TOKEN}  --discovery-token-ca-cert-hash ${CA_CERT_HASH}  --control-plane --certificate-key areyoukidingme
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
   exit 1
fi
mkdir -p $HOME/.kube
if [[ -f $HOME/.kube/config ]];then rm -f $HOME/.kube/config;fi
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
EOF
   install_binary
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
${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}
/tmp/kubeadm join ${KUBEADM_ARGS} ${controlPlaneEndpoint} --token ${K8S_TOKEN}  --discovery-token-ca-cert-hash ${CA_CERT_HASH}
if [[ \$? -eq 0  ]];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
   exit 1
fi
rm -f /tmp/kubeadm
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
              Join
              exit 0
              shift 3;;
          "Init" )
              if [[ "$#" -gt 1 ]]; then
                SERVER_URL="$2"
              fi
              Init
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
