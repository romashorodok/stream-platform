package netutils

import (
	"fmt"
	"net"

	"golang.org/x/exp/slices"
)

func GetLocalIPAddresses(includeLoopback bool, preferredInterfaces []string) ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	loopBacks := make([]string, 0)
	addresses := make([]string, 0)
	for _, iface := range ifaces {
		if len(preferredInterfaces) != 0 && !slices.Contains(preferredInterfaces, iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch typedAddr := addr.(type) {
			case *net.IPNet:
				ip = typedAddr.IP.To4()
			case *net.IPAddr:
				ip = typedAddr.IP.To4()
			default:
				continue
			}
			if ip == nil {
				continue
			}
			if ip.IsLoopback() {
				loopBacks = append(loopBacks, ip.String())
			} else {
				addresses = append(addresses, ip.String())
			}
		}
	}
	if includeLoopback {
		addresses = append(addresses, loopBacks...)
	}
	if len(addresses) > 0 {
		return addresses, nil
	}
	if len(loopBacks) > 0 {
		return loopBacks, nil
	}
	return nil, fmt.Errorf("could not find local IP address")
}
