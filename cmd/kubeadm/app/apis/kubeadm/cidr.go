/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package kubeadm

import (
	"fmt"
	"net"
	"regexp"
)

type Order int

const (
	Reverse Order = iota
	Positive
)


var RegRule = regexp.MustCompile("lo|cali.*|veth.*|tunl.*")

func IsLocalNetwork(name string) bool {
	return !RegRule.MatchString(name)
}

func getHostCIDR() map[string]string {
	ret := map[string]string{}
	interfaces, _ := net.Interfaces()
	for _, intf := range interfaces {
		if !IsLocalNetwork(intf.Name) {
			continue
		}
		addrs, err := intf.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, ipnet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() != nil {
				ipnet.Mask = net.CIDRMask(16, 32)
				ipnet.IP = ipnet.IP.Mask(ipnet.Mask)
				ret[ipnet.String()] = "ipv4"
			}
		}
	}
	return ret
}

// kubeadm init --pod-network-cidr
// Specify range of IP addresses for the pod network;
// if set, the control plane will automatically allocate CIDRs for every node
// AcquirePodCIDR for --pod-network-cidr
// TODO: FIXME to support ipv6
func AcquirePodCIDR(prefix, from, to int, order Order) string {
	ret := "172.31.0.0/16"
	hostCIDR := getHostCIDR()
	switch order {
	case Reverse:
		for i := to; i >= from && i <= to; i-- {
			cidr := fmt.Sprintf("%d.%d.0.0/16", prefix, i)
			if _, ok := hostCIDR[cidr]; !ok {
				//fmt.Printf("Acquire pod cidr [%s]. \n",cidr)
				return cidr
			}
		}
	case Positive:
		for i := from; i >= from && i <= to; i++ {
			cidr := fmt.Sprintf("%d.%d.0.0/16", prefix, i)
			if _, ok := hostCIDR[cidr]; !ok {
				//fmt.Printf("Acquire pod cidr [%s]. \n",cidr)
				return cidr
			}
		}
	}
	return ret
}
