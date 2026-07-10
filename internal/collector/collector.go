package collector

import (
	"fmt"
	"sort"

	gnet "github.com/shirou/gopsutil/v3/net"
)

type Snapshot struct {
	Interface string
	RxBytes   uint64
	TxBytes   uint64
}

type Collector struct {
	iface string // "auto" or specific name
}

func New(iface string) *Collector {
	return &Collector{iface: iface}
}

func (c *Collector) Read() (Snapshot, error) {
	stats, err := gnet.IOCounters(true)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collector: IOCounters: %w", err)
	}

	if c.iface != "auto" && c.iface != "" {
		for _, s := range stats {
			if s.Name == c.iface {
				return Snapshot{
					Interface: s.Name,
					RxBytes:   s.BytesRecv,
					TxBytes:   s.BytesSent,
				}, nil
			}
		}
		return Snapshot{}, fmt.Errorf("collector: interface %q not found", c.iface)
	}

	best := pickBest(stats)
	if best == nil {
		return Snapshot{}, fmt.Errorf("collector: no usable network interface found")
	}
	return Snapshot{
		Interface: best.Name,
		RxBytes:   best.BytesRecv,
		TxBytes:   best.BytesSent,
	}, nil
}

func Interfaces() ([]string, error) {
	stats, err := gnet.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("collector: IOCounters: %w", err)
	}
	names := make([]string, 0, len(stats))
	for _, s := range stats {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names, nil
}

type InterfaceDetail struct {
	Name         string
	HardwareAddr string
	Addrs        []string
	IsUp         bool
	Mtu          int
}

func InterfaceDetails(name string) (*InterfaceDetail, error) {
	interfaces, err := gnet.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("collector: Interfaces: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Name == name {
			detail := &InterfaceDetail{
				Name:         iface.Name,
				HardwareAddr: iface.HardwareAddr,
				IsUp:         len(iface.Flags) > 0,
				Mtu:          iface.MTU,
			}
			for _, addr := range iface.Addrs {
				detail.Addrs = append(detail.Addrs, addr.Addr)
			}
			for _, flag := range iface.Flags {
				if flag == "up" {
					detail.IsUp = true
					break
				}
			}
			return detail, nil
		}
	}
	return nil, fmt.Errorf("collector: interface %q not found", name)
}

func pickBest(stats []gnet.IOCountersStat) *gnet.IOCountersStat {
	var best *gnet.IOCountersStat
	var bestTotal uint64

	for i := range stats {
		s := &stats[i]
		if isLoopback(s.Name) {
			continue
		}
		total := s.BytesRecv + s.BytesSent
		if best == nil || total > bestTotal {
			best = s
			bestTotal = total
		}
	}
	return best
}

func isLoopback(name string) bool {
	switch name {
	case "lo", "lo0":
		return true
	}
	prefixes := []string{"docker", "br-", "veth", "virbr", "vmnet", "vbox"}
	for _, p := range prefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}
