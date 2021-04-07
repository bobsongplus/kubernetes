load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["network.go"],
    importpath = "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network",
    visibility = ["//visibility:public"],
    deps = [
        "//cmd/kubeadm/app/apis/kubeadm:go_default_library",
        "//cmd/kubeadm/app/constants:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/calico:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/canal:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/flannel:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/macvlan:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/multus:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/ovn:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/ovnkube:go_default_library",
        "//cmd/kubeadm/app/phases/addons/network/weavenet:go_default_library",
        "//staging/src/k8s.io/client-go/kubernetes:go_default_library",
    ],
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [
        ":package-srcs",
        "//cmd/kubeadm/app/phases/addons/network/calico:all-srcs",
        "//cmd/kubeadm/app/phases/addons/network/canal:all-srcs",
        "//cmd/kubeadm/app/phases/addons/network/flannel:all-srcs",
        "//cmd/kubeadm/app/phases/addons/network/macvlan:all-srcs",
        "//cmd/kubeadm/app/phases/addons/network/multus:all-srcs",
        "//cmd/kubeadm/app/phases/addons/network/ovn:all-srcs",
        "//cmd/kubeadm/app/phases/addons/network/ovnkube:all-srcs",
        "//cmd/kubeadm/app/phases/addons/network/weavenet:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
