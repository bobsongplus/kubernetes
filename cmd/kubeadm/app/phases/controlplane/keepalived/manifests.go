package keepalived

const (
	Version = "2.2.0"

	DefaultKeepalivedCfg = "keepalived.conf.tmpl"
	DefaultKeepalived  = "keepalived.yaml"

	KeepalivedCfg = `
global_defs {
   script_user root
   enable_script_security
   router_id control-plane
   vrrp_garp_master_delay 2
   vrrp_garp_master_repeat 3
   vrrp_garp_master_refresh 30
   vrrp_garp_interval 0.001
   vrrp_gna_interval 0.000001
   vrrp_no_swap
   checker_no_swap
}
vrrp_script haproxy {
    script "/etc/confd/templates/probe.sh"
    interval 2
    timeout 1
    rise 1
    fall 3
    user root root
}
vrrp_instance master {
    state BACKUP
    interface [[.Interface]]
    virtual_router_id 191
    priority [[.Priority]]
    advert_int 1
    authentication {
        auth_type PASS
        auth_pass Dream001
    }
    track_script {
        haproxy
    }
    virtual_ipaddress {
        [[.VIP]]
    }
   unicast_src_ip [[.HostIP]]
   unicast_peer { {{range $s := ls "/masterleases"}}
     {{ if ne $s "[[.HostIP]]" }}{{$s}}{{end}}{{end}}
   }

}
`

	Keepalived = `
kind: Pod
apiVersion: v1
metadata:
  name: kube-keepalived
  namespace: kube-system
  labels:
    k8s-app: keepalived
    tier: control-plane
spec:
  hostNetwork: true
  priorityClassName: system-cluster-critical
  containers:
    - name: keepalived
      image: {{ .ImageRepository }}/keepalived-{{.Arch}}:{{.Version}}
      volumeMounts:
        - mountPath: /etc/confd/templates
          name: config
        - mountPath: /etc/kubernetes/pki
          name: pki
      resources:
        requests:
          cpu: 100m
      securityContext:
        privileged: true
        capabilities:
          add: [NET_ADMIN,NET_BROADCAST,NET_RAW]
  volumes:
    - name: config
      hostPath:
        path: /etc/kubernetes/keepalived
        type: DirectoryOrCreate
    - name: pki
      hostPath:
        path: /etc/kubernetes/pki
        type: DirectoryOrCreate
`
)

