package system

import (
	"fmt"
	"net"
)

const (
	Major = 2
	Minor = 0
	Patch = 2
)

func IPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""

}

// Examples:
//
//	v1.20.1
func NodeVersion() string {
	v := fmt.Sprintf("v%d.%d.%d", Major, Minor, Patch)
	return v
}
