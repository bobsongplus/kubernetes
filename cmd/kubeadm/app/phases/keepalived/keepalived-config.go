/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2020 TenxCloud. All Rights Reserved.
 * 2020  @author tenxcloud
 */
package keepalived

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"k8s.io/klog/v2"
)

const (
	DefaultKeepalivedDir    = "keepalived"
	DefaultKeepalivedConfig = "keepalived.conf"
)

var keepalivedTmpl = `
global_defs {
   script_user root
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
    script "/container/service/keepalived/assets/probe.sh"
    interval 2
    timeout 1
    rise 1
    fall 3
    user root root
}
vrrp_instance master {
    state BACKUP
    interface {{.Interface }}
    virtual_router_id 191
    priority 191
    advert_int 1
    authentication {
        auth_type PASS
        auth_pass Dream001
    }
    track_script {
        haproxy
    }
    virtual_ipaddress {
        {{ .VIP }}
    }
   unicast_src_ip {{ .HostIP }}
   unicast_peer { {{ range .Peers }}
    {{ . }}{{ end }}
   }
}`

type AddressFamily uint

const (
	familyIPv4 AddressFamily = 4
	familyIPv6 AddressFamily = 6
)

const (
	ipv4RouteFile = "/proc/net/route"
	ipv6RouteFile = "/proc/net/ipv6_route"
)

type Route struct {
	Interface   string
	Destination net.IP
	Gateway     net.IP
	Family      AddressFamily
}

type RouteFile struct {
	name  string
	parse func(input io.Reader) ([]Route, error)
}

// noRoutesError can be returned by ChooseBindAddress() in case of no routes
type noRoutesError struct {
	message string
}

type keepalived struct {
	HostIP    string
	Interface string
	VIP       string
	Peers     []string
}

func (e noRoutesError) Error() string {
	return e.message
}

// IsNoRoutesError checks if an error is of type noRoutesError
func IsNoRoutesError(err error) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	case noRoutesError:
		return true
	default:
		return false
	}
}

var (
	v4File = RouteFile{name: ipv4RouteFile, parse: getIPv4DefaultRoutes}
	v6File = RouteFile{name: ipv6RouteFile, parse: getIPv6DefaultRoutes}
)

func (rf RouteFile) extract() ([]Route, error) {
	file, err := os.Open(rf.name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return rf.parse(file)
}

// getIPv4DefaultRoutes obtains the IPv4 routes, and filters out non-default routes.
func getIPv4DefaultRoutes(input io.Reader) ([]Route, error) {
	routes := []Route{}
	scanner := bufio.NewReader(input)
	for {
		line, err := scanner.ReadString('\n')
		if err == io.EOF {
			break
		}
		//ignore the headers in the route info
		if strings.HasPrefix(line, "Iface") {
			continue
		}
		fields := strings.Fields(line)
		// Interested in fields:
		//  0 - interface name
		//  1 - destination address
		//  2 - gateway
		dest, err := parseIP(fields[1], familyIPv4)
		if err != nil {
			return nil, err
		}
		gw, err := parseIP(fields[2], familyIPv4)
		if err != nil {
			return nil, err
		}
		if !dest.Equal(net.IPv4zero) {
			continue
		}
		routes = append(routes, Route{
			Interface:   fields[0],
			Destination: dest,
			Gateway:     gw,
			Family:      familyIPv4,
		})
	}
	return routes, nil
}

func getIPv6DefaultRoutes(input io.Reader) ([]Route, error) {
	routes := []Route{}
	scanner := bufio.NewReader(input)
	for {
		line, err := scanner.ReadString('\n')
		if err == io.EOF {
			break
		}
		fields := strings.Fields(line)
		// Interested in fields:
		//  0 - destination address
		//  4 - gateway
		//  9 - interface name
		dest, err := parseIP(fields[0], familyIPv6)
		if err != nil {
			return nil, err
		}
		gw, err := parseIP(fields[4], familyIPv6)
		if err != nil {
			return nil, err
		}
		if !dest.Equal(net.IPv6zero) {
			continue
		}
		if gw.Equal(net.IPv6zero) {
			continue // loopback
		}
		routes = append(routes, Route{
			Interface:   fields[9],
			Destination: dest,
			Gateway:     gw,
			Family:      familyIPv6,
		})
	}
	return routes, nil
}

// parseIP takes the hex IP address string from route file and converts it
// to a net.IP address. For IPv4, the value must be converted to big endian.
func parseIP(str string, family AddressFamily) (net.IP, error) {
	if str == "" {
		return nil, fmt.Errorf("input is nil")
	}
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	if family == familyIPv4 {
		if len(bytes) != net.IPv4len {
			return nil, fmt.Errorf("invalid IPv4 address in route")
		}
		return net.IP([]byte{bytes[3], bytes[2], bytes[1], bytes[0]}), nil
	}
	// Must be IPv6
	if len(bytes) != net.IPv6len {
		return nil, fmt.Errorf("invalid IPv6 address in route")
	}
	return net.IP(bytes), nil
}

func isInterfaceUp(intf *net.Interface) bool {
	if intf == nil {
		return false
	}
	if intf.Flags&net.FlagUp != 0 {
		klog.V(4).Infof("Interface %v is up", intf.Name)
		return true
	}
	return false
}

func isLoopbackOrPointToPoint(intf *net.Interface) bool {
	return intf.Flags&(net.FlagLoopback|net.FlagPointToPoint) != 0
}

// getMatchingGlobalIP returns the first valid global unicast address of the given
// 'family' from the list of 'addrs'.
func getMatchingGlobalIP(addrs []net.Addr, family AddressFamily) (net.IP, error) {
	if len(addrs) > 0 {
		for i := range addrs {
			klog.V(4).Infof("Checking addr  %s.", addrs[i].String())
			ip, _, err := net.ParseCIDR(addrs[i].String())
			if err != nil {
				return nil, err
			}
			if memberOf(ip, family) {
				if ip.IsGlobalUnicast() {
					klog.V(4).Infof("IP found %v", ip)
					return ip, nil
				} else {
					klog.V(4).Infof("Non-global unicast address found %v", ip)
				}
			} else {
				klog.V(4).Infof("%v is not an IPv%d address", ip, int(family))
			}

		}
	}
	return nil, nil
}

// getIPFromInterface gets the IPs on an interface and returns a global unicast address, if any. The
// interface must be up, the IP must in the family requested, and the IP must be a global unicast address.
func getIPFromInterface(intfName string, forFamily AddressFamily, nw networkInterfacer) (net.IP, error) {
	intf, err := nw.InterfaceByName(intfName)
	if err != nil {
		return nil, err
	}
	if isInterfaceUp(intf) {
		addrs, err := nw.Addrs(intf)
		if err != nil {
			return nil, err
		}
		klog.V(4).Infof("Interface %q has %d addresses :%v.", intfName, len(addrs), addrs)
		matchingIP, err := getMatchingGlobalIP(addrs, forFamily)
		if err != nil {
			return nil, err
		}
		if matchingIP != nil {
			klog.V(4).Infof("Found valid IPv%d address %v for interface %q.", int(forFamily), matchingIP, intfName)
			return matchingIP, nil
		}
	}
	return nil, nil
}

// memberOF tells if the IP is of the desired family. Used for checking interface addresses.
func memberOf(ip net.IP, family AddressFamily) bool {
	if ip.To4() != nil {
		return family == familyIPv4
	} else {
		return family == familyIPv6
	}
}

// chooseIPFromHostInterfaces looks at all system interfaces, trying to find one that is up that
// has a global unicast address (non-loopback, non-link local, non-point2point), and returns the IP.
// Searches for IPv4 addresses, and then IPv6 addresses.
func chooseIPFromHostInterfaces(nw networkInterfacer) (net.IP, string, error) {
	intfs, err := nw.Interfaces()
	if err != nil {
		return nil, "", err
	}
	if len(intfs) == 0 {
		return nil, "", fmt.Errorf("no interfaces found on host")
	}
	for _, family := range []AddressFamily{familyIPv4, familyIPv6} {
		klog.V(4).Infof("Looking for system interface with a global IPv%d address", uint(family))
		for _, intf := range intfs {
			if !isInterfaceUp(&intf) {
				klog.V(4).Infof("Skipping: down interface %q", intf.Name)
				continue
			}
			if isLoopbackOrPointToPoint(&intf) {
				klog.V(4).Infof("Skipping: LB or P2P interface %q", intf.Name)
				continue
			}
			addrs, err := nw.Addrs(&intf)
			if err != nil {
				return nil, "", err
			}
			if len(addrs) == 0 {
				klog.V(4).Infof("Skipping: no addresses on interface %q", intf.Name)
				continue
			}
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					return nil, "", fmt.Errorf("Unable to parse CIDR for interface %q: %s", intf.Name, err)
				}
				if !memberOf(ip, family) {
					klog.V(4).Infof("Skipping: no address family match for %q on interface %q.", ip, intf.Name)
					continue
				}
				// TODO: Decide if should open up to allow IPv6 LLAs in future.
				if !ip.IsGlobalUnicast() {
					klog.V(4).Infof("Skipping: non-global address %q on interface %q.", ip, intf.Name)
					continue
				}
				klog.V(4).Infof("Found global unicast address %q on interface %q.", ip, intf.Name)
				return ip, intf.Name, nil
			}
		}
	}
	return nil, "", fmt.Errorf("no acceptable interface with global unicast address found on host")
}

// ChooseHostInterface is a method used fetch an IP and Interface for a daemon.
// If there is no routing info file, it will choose a global IP from the system
// interfaces. Otherwise, it will use IPv4 and IPv6 route information to return the
// IP of the interface with a gateway on it (with priority given to IPv4). For a node
// with no internet connection, it returns error.
func ChooseHostInterface() (net.IP, string, error) {
	var nw networkInterfacer = networkInterface{}
	if _, err := os.Stat(ipv4RouteFile); os.IsNotExist(err) {
		return chooseIPFromHostInterfaces(nw)
	}
	routes, err := getAllDefaultRoutes()
	if err != nil {
		return nil, "", err
	}
	return chooseHostInterfaceFromRoute(routes, nw)
}

// networkInterfacer defines an interface for several net library functions. Production
// code will forward to net library functions, and unit tests will override the methods
// for testing purposes.
type networkInterfacer interface {
	InterfaceByName(intfName string) (*net.Interface, error)
	Addrs(intf *net.Interface) ([]net.Addr, error)
	Interfaces() ([]net.Interface, error)
}

// networkInterface implements the networkInterfacer interface for production code, just
// wrapping the underlying net library function calls.
type networkInterface struct{}

func (_ networkInterface) InterfaceByName(intfName string) (*net.Interface, error) {
	return net.InterfaceByName(intfName)
}

func (_ networkInterface) Addrs(intf *net.Interface) ([]net.Addr, error) {
	return intf.Addrs()
}

func (_ networkInterface) Interfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// getAllDefaultRoutes obtains IPv4 and IPv6 default routes on the node. If unable
// to read the IPv4 routing info file, we return an error. If unable to read the IPv6
// routing info file (which is optional), we'll just use the IPv4 route information.
// Using all the routing info, if no default routes are found, an error is returned.
func getAllDefaultRoutes() ([]Route, error) {
	routes, err := v4File.extract()
	if err != nil {
		return nil, err
	}
	v6Routes, _ := v6File.extract()
	routes = append(routes, v6Routes...)
	if len(routes) == 0 {
		return nil, noRoutesError{
			message: fmt.Sprintf("no default routes found in %q or %q", v4File.name, v6File.name),
		}
	}
	return routes, nil
}

// chooseHostInterfaceFromRoute cycles through each default route provided, looking for a
// global IP address from the interface for the route. Will first look all each IPv4 route for
// an IPv4 IP, and then will look at each IPv6 route for an IPv6 IP.
func chooseHostInterfaceFromRoute(routes []Route, nw networkInterfacer) (net.IP, string, error) {
	for _, family := range []AddressFamily{familyIPv4, familyIPv6} {
		klog.V(4).Infof("Looking for default routes with IPv%d addresses", uint(family))
		for _, route := range routes {
			if route.Family != family {
				continue
			}
			klog.V(4).Infof("Default route transits interface %q", route.Interface)
			finalIP, err := getIPFromInterface(route.Interface, family, nw)
			if err != nil {
				return nil, "", err
			}
			if finalIP != nil {
				klog.V(4).Infof("Found active IP %v ", finalIP)
				return finalIP, route.Interface, nil
			}
		}
	}
	klog.V(4).Infof("No active IP found by looking at default routes")
	return nil, "", fmt.Errorf("unable to select an IP from default routes")
}

// VerifyServerBindAddress can be used to verify if a bind address for the API Server is 0.0.0.0,
// in which case this address is not valid and should not be used.
func VerifyServerBindAddress(address string) error {
	ip := net.ParseIP(address)
	if ip == nil {
		return fmt.Errorf("cannot parse IP address: %s", address)
	}
	if !ip.IsGlobalUnicast() {
		return fmt.Errorf("cannot use %q as the bind address for the API Server", address)
	}
	return nil
}

// VerifyServerBindAddress can be used to verify if a bind address for the API Server is 0.0.0.0,
// in which case this address is not valid and should not be used.
func VerifyMcastGroupAddress(address string) error {
	ip := net.ParseIP(address)
	if ip == nil {
		return fmt.Errorf("cannot parse IP address: %s", address)
	}
	if !ip.IsMulticast() {
		return fmt.Errorf("cannot use %q as the McastGroup address for KeepAlived", address)
	}
	return nil
}
