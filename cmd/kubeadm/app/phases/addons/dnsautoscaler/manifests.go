/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package dnsautoscaler

/*
 *
 * DNS Horizontal Autoscaler
 *
 * DNS Horizontal Autoscaler enables horizontal autoscaling feature for DNS service in Kubernetes clusters.
 * This autoscaler runs as a Deployment. It collects cluster status from the APIServer,
 * horizontally scales the number of DNS backends based on demand.
 * Autoscaling parameters could be tuned by modifying the kube-dns-autoscaler ConfigMap in kube-system namespace.
 *
 * gcr.io/google-containers/cluster-proportional-autoscaler-amd64:1.6.0
 * k8s.gcr.io/cpa/cluster-proportional-autoscaler-amd64:1.8.3
 *
 * http://kubernetes.io/docs/tasks/administer-cluster/dns-horizontal-autoscaling/
 * https://github.com/kubernetes-incubator/cluster-proportional-autoscaler/
 * https://github.com/kubernetes-sigs/cluster-proportional-autoscaler
 * https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/dns-horizontal-autoscaler
 *
 */

const (
	CoreDnsAutoscalerVersion = "1.8.3"

	CoreDnsAutoscaler = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns-autoscaler
  namespace: kube-system
  labels:
    k8s-app: coredns-autoscaler
    kubernetes.io/cluster-service: "true"
spec:
  selector:
    matchLabels:
      k8s-app: coredns-autoscaler
  template:
    metadata:
      labels:
        k8s-app: coredns-autoscaler
    spec:
      priorityClassName: system-cluster-critical
      containers:
      - name: autoscaler
        image: {{ .ImageRepository }}/cluster-proportional-autoscaler-{{ .Arch }}:{{ .Version }}
        resources:
            requests:
                cpu: "20m"
                memory: "10Mi"
        command:
          - /cluster-proportional-autoscaler
          - --namespace=kube-system
          - --configmap=coredns-autoscaler
          - --target={{.Target}}
          - --default-params={"linear":{"coresPerReplica":256,"nodesPerReplica":16,"preventSinglePointFailure":true,"includeUnschedulableNodes":true}}
          - --logtostderr=true
          - --v=2
      tolerations:
      - operator: Exists
      serviceAccountName: coredns-autoscaler
`

	// for kube-dns-autoscaler
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns-autoscaler
  namespace: kube-system
`

	ClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:coredns-autoscaler
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get","list","watch"]
  - apiGroups: ["apps"]
    resources: ["deployments/scale", "replicasets/scale"]
    verbs: ["get", "update"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "create"]
`

	ClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:coredns-autoscaler
subjects:
  - kind: ServiceAccount
    name: coredns-autoscaler
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: system:coredns-autoscaler
  apiGroup: rbac.authorization.k8s.io
`
)
