/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package serviceproxy

const (
	TenxProxyVersion       = "v4.1.0"
	HarpoxyExporterVersion = "v0.8.0"

	TenxProxyTemplate = `
apiVersion: v1
data:
  haproxy.tpl: |
    # Licensed Materials - Property of tenxcloud.com
    # (C) Copyright 2017 TenxCloud. All Rights Reserved.
    # 2017-07-20  @author lizhen

    global
        log 127.0.0.1 local2
        chroot /var/lib/haproxy
        stats socket /run/haproxy/admin.sock mode 660 level admin
        stats timeout 600s
        ulimit-n 1500000
        maxconn 2000000
        user haproxy
        group haproxy
        daemon
        tune.ssl.default-dh-param 2048
        ssl-default-bind-options no-sslv3 no-tlsv10
        ssl-default-bind-ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-DSS-AES128-GCM-SHA256:kEDH+AESGCM:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA:ECDHE-ECDSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES128-SHA:DHE-DSS-AES128-SHA256:DHE-RSA-AES256-SHA256:DHE-DSS-AES256-SHA:DHE-RSA-AES256-SHA:!aNULL:!eNULL:!EXPORT:!DES:!RC4:!3DES:!MD5:!PSK

    defaults
        mode                    http
        log                     global
        option                  dontlognull
        option                  http-server-close
        option                  redispatch
        retries                 3
        timeout http-request    5000
        timeout queue           30000
        timeout connect         2000
        timeout client          3600000
        timeout server          3600000
        timeout http-keep-alive 500
        timeout check           5000
        maxconn                 50000

    listen stats
        bind *:8889
        mode http
        stats uri /tenx-stats
        stats realm Haproxy\ Statistics
        stats auth tenxcloud:haproxy-agent

    {{with .DefaultHTTP}}
    listen defaulthttp
        bind {{$.PublicIP}}:80
        mode http
        option forwardfor       except 127.0.0.0/8
        errorfile 503 /etc/haproxy/errors/503.http{{range .Redirect}}
        redirect scheme https code 301 if { hdr(Host) -i {{range .DomainNames}} {{.}}{{end}} } !{ ssl_fc }{{end}}{{range .Domains}}
        acl {{.BackendName}} hdr(host) -i {{range .DomainNames}} {{.}}{{end}}
        use_backend {{.BackendName}} if {{.BackendName}}{{end}}{{end}}

    {{with .FrontendLB}}
    frontend LB
        mode http
        option forwardfor       except 127.0.0.0/8
        errorfile 503 /etc/haproxy/errors/503.http
        bind {{$.PublicIP}}:443 ssl crt {{.DefaultSSLCert}}{{range .SSLCerts}} crt {{.}}{{end}}{{range .Domains}}
        acl {{.BackendName}} hdr(host) -i {{range .DomainNames}} {{.}}{{end}}
        use_backend {{.BackendName}} if {{.BackendName}} { ssl_fc_sni{{range .DomainNames}} {{.}}{{end}} }{{end}}{{end}}

    {{with .Listen}}{{range .}}
    listen {{.DomainName}}
        bind {{$.PublicIP}}:{{.PublicPort}}
        mode tcp
        balance roundrobin{{$port := .Port}}{{range .Pods}}
        server {{.Name}} {{.IP}}:{{$port}} maxconn 5000{{end}}{{end}}{{end}}

    {{with .Backend}}{{range .}}
    backend {{.BackendName}}{{$port := .Port}}{{range .Pods}}
        server {{.Name}} {{.IP}}:{{$port}} cookie {{.Name}} check maxconn 5000{{end}}{{end}}{{end}}
kind: ConfigMap
metadata:
  name: service-proxy-template
  namespace: kube-system
`
	TenxProxyDomainConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-config
  namespace: kube-system
data:
  domain.json: '{"externalip":"","domain":""}'
`
	TenxProxyCertsConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-certs
  namespace: kube-system
data:
`

	TenxProxyDaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    name: service-proxy
  name: service-proxy
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: service-proxy
  template:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9101"
      labels:
        name: service-proxy
    spec:
      serviceAccountName: service-proxy
      containers:
      - command:
        - /run.sh
        - --plugins=tenx-proxy --watch=watchsrvs --emailReceiver=weiwei@tenxcloud.com
          --config=/etc/tenx/domain.json
        image: {{ .ImageRepository }}/tenx-proxy-{{ .Arch }}:{{ .TenxProxyVersion }}
        imagePullPolicy: IfNotPresent
        name: service-proxy
        volumeMounts:
        - mountPath: /var/run/docker.sock
          name: docker-sock
        - mountPath: /etc/tenx/
          name: kube-config
        - mountPath: /etc/sslkeys/certs
          name: kube-cert
        - mountPath: /etc/default/hafolder/haproxy.tpl
          name: service-proxy-template
          subPath: haproxy.tpl
        - mountPath: /run/haproxy
          name: haproxy-sock
      - command:
        - sh
        - -c
        - sleep 10 && haproxy_exporter --haproxy.scrape-uri=unix:/run/haproxy/admin.sock
        image: {{ .ImageRepository }}/haproxy-exporter-{{ .Arch }}:{{.HarpoxyExporterVersion}}
        imagePullPolicy: IfNotPresent
        name: exporter
        ports:
        - containerPort: 9101
          hostPort: 9101
          name: scrape
          protocol: TCP
        resources: {}
        volumeMounts:
        - mountPath: /run/haproxy
          name: haproxy-sock
      dnsPolicy: ClusterFirst
      hostNetwork: true
      nodeSelector:
        role: proxy
      restartPolicy: Always
      volumes:
      - emptyDir: {}
        name: docker-sock
      - hostPath:
          path: /var/run/docker.sock
        name: config-volume
      - configMap:
          defaultMode: 420
          name: kube-config
        name: kube-config
      - configMap:
          defaultMode: 420
          name: service-proxy-template
        name: service-proxy-template
      - configMap:
          defaultMode: 420
          name: kube-certs
        name: kube-cert
      - emptyDir: {}
        name: haproxy-sock
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: role
                operator: In
                values:
                - proxy
`

	// for service-proxy
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: service-proxy
  namespace: kube-system
`

	ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:service-proxy
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: service-proxy
  namespace: kube-system
`
)
