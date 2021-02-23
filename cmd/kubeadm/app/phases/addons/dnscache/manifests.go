package dnscache

const (

	// https://github.com/kubernetes/dns/tree/master/cmd/node-cache
	// https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/0030-nodelocal-dns-cache.md

	DnsCacheVersion  = "1.15.10"

	CoreDnsCache =`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: coredns-cache
  namespace: kube-system
  labels:
    k8s-app: coredns-cache
spec:
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 10%
  selector:
    matchLabels:
      k8s-app: coredns-cache
  template:
    metadata:
       labels:
         k8s-app: coredns-cache
       annotations:
         prometheus.io/port: "9253"
         prometheus.io/scrape: "true"          
    spec:
      priorityClassName: system-node-critical
      hostNetwork: true
      dnsPolicy: Default
      tolerations:
      - effect: "NoExecute"
        operator: "Exists"
      - effect: "NoSchedule"
        operator: "Exists"
      nodeSelector:
        beta.kubernetes.io/os: linux
        beta.kubernetes.io/arch: {{ .Arch }}
      containers:
      - name: cache
        image: {{ .ImageRepository }}/k8s-dns-node-cache-{{ .Arch }}:{{ .Version }}
        resources:
          requests:
            cpu: 200m
            memory: 512Mi
          limits:
            cpu: 200m
            memory: 512Mi
        args: [ "-localip", "{{ .LocalIP }}", "-conf", "/etc/Corefile", "-interfacename", "dns"]
        securityContext:
          privileged: true
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
        - containerPort: 9253
          name: metrics
          protocol: TCP
        livenessProbe:
          httpGet:
            host: {{ .LocalDNSAddress }}
            path: /health
            port: 8080
          initialDelaySeconds: 60
          timeoutSeconds: 5
        volumeMounts:
        - mountPath: /run/xtables.lock
          name: xtables-lock
          readOnly: false
        - name: config-volume
          mountPath: /etc/coredns
        - name: kube-dns-config
          mountPath: /etc/kube-dns
      volumes:
      - name: xtables-lock
        hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
      - name: kube-dns-config
        configMap:
          name: kube-dns
          optional: true
      - name: config-volume
        configMap:
          name: coredns-cache
          items:
            - key: Corefile
              path: Corefile.base
`

	ConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns-cache
  namespace: kube-system
data:
  Corefile: |
    {{ .DNSDomain }}:53 {
        errors
        cache {
           success 9984 30
           denial 9984 5
        }
        reload
        loop
        bind {{ .LocalDNSAddress }} {{ .DNSServerAddress }}
        forward . __PILLAR__CLUSTER__DNS__  {
           force_tcp
        }
        prometheus :9253
        health {{ .LocalDNSAddress }}:8080
        }
    in-addr.arpa:53 {
        errors
        cache 30
        reload
        loop
        bind {{ .LocalDNSAddress }} {{ .DNSServerAddress }}
        forward .  __PILLAR__CLUSTER__DNS__  {
           force_tcp
        }
        prometheus :9253
        }
    ip6.arpa:53 {
        errors
        cache 30
        reload
        loop
        bind {{ .LocalDNSAddress }} {{ .DNSServerAddress }}
        forward .  __PILLAR__CLUSTER__DNS__  {
           force_tcp
        }
        prometheus :9253
        }
    .:53 {
        errors
        cache 30
        reload
        loop
        bind {{ .LocalDNSAddress }} {{ .DNSServerAddress }}
        forward . __PILLAR__UPSTREAM__SERVERS__ {
          force_tcp
        }
        prometheus :9253
        }
`

)
