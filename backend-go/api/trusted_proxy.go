package api

import (
	"fmt"
	"net"
	"strings"
)

func validateTrustedProxyList(proxies []string) error {
	for _, raw := range proxies {
		proxy := strings.TrimSpace(raw)
		if proxy == "" {
			return fmt.Errorf("trusted proxy entry cannot be empty")
		}
		if strings.Contains(proxy, "/") {
			_, network, err := net.ParseCIDR(proxy)
			if err != nil {
				return fmt.Errorf("invalid trusted proxy CIDR %q", proxy)
			}
			ones, _ := network.Mask.Size()
			if ones == 0 {
				return fmt.Errorf("refusing wildcard trusted proxy CIDR %q", proxy)
			}
			continue
		}
		if net.ParseIP(proxy) == nil {
			return fmt.Errorf("invalid trusted proxy IP %q", proxy)
		}
	}
	return nil
}
