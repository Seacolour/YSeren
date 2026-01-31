package main

import (
	"net"
	"sort"
)

// ListLANIPv4 返回所有可用于局域网访问的 IPv4（排除 loopback / link-local / 虚拟无效地址）。
// 典型返回：192.168.x.x / 10.x.x.x / 172.16~31.x.x
func ListLANIPv4() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var ips []string
	seen := map[string]struct{}{}

	for _, ifc := range ifaces {
		// 跳过 down 或 loopback 网卡
		if (ifc.Flags&net.FlagUp) == 0 || (ifc.Flags&net.FlagLoopback) != 0 {
			continue
		}

		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := ipFromAddr(addr)
			if ip == nil {
				continue
			}
			if ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			// 过滤 169.254.0.0/16 (link-local)
			if ip4[0] == 169 && ip4[1] == 254 {
				continue
			}
			// 只保留私有网段（避免把公网 IP 打出来）
			if !isPrivateIPv4(ip4) {
				continue
			}
			s := ip4.String()
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			ips = append(ips, s)
		}
	}

	sort.Strings(ips)
	return ips
}

func ipFromAddr(a net.Addr) net.IP {
	switch v := a.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}

func isPrivateIPv4(ip net.IP) bool {
	// RFC1918: 10.0.0.0/8
	if ip[0] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}
	return false
}
