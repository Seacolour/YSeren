package runtime

import (
	"net"
	"sort"
)

// ListLANIPv4 返回可用于局域网访问的 RFC1918 IPv4 地址。
func ListLANIPv4() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	addresses := make([]string, 0)
	seen := make(map[string]struct{})
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		interfaceAddresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range interfaceAddresses {
			ip := ipFromAddress(address)
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ipv4 := ip.To4()
			if ipv4 == nil || ipv4[0] == 169 && ipv4[1] == 254 || !isPrivateIPv4(ipv4) {
				continue
			}
			value := ipv4.String()
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			addresses = append(addresses, value)
		}
	}
	sort.Strings(addresses)
	return addresses
}

func ipFromAddress(address net.Addr) net.IP {
	switch value := address.(type) {
	case *net.IPNet:
		return value.IP
	case *net.IPAddr:
		return value.IP
	default:
		return nil
	}
}

func isPrivateIPv4(ip net.IP) bool {
	return ip[0] == 10 ||
		ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 ||
		ip[0] == 192 && ip[1] == 168
}
