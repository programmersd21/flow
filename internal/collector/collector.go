package collector

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	gnet "github.com/shirou/gopsutil/v3/net"
)

// Snapshot is a point-in-time reading of one interface's byte counters.
type Snapshot struct {
	Interface string
	RxBytes   uint64
	TxBytes   uint64
}

// Reader is the counter source the sampler depends on. Keeping it an interface
// lets the sampler be tested against deterministic counter sequences.
type Reader interface {
	Read() (Snapshot, error)
}

type counter struct {
	bytes uint64
	at    time.Time
}

// Collector reads interface byte counters.
//
// Two modes:
//   - a fixed interface name, which simply reports that interface
//   - "auto", which picks the interface currently carrying the most traffic
type Collector struct {
	mu    sync.Mutex
	iface string

	// auto-mode state. prev holds the previous reading of every interface so a
	// per-interface rate can be derived; current is the chosen interface.
	prev    map[string]counter
	current string
	lastAt  time.Time
}

// switchRatio is how much busier a challenger must be before it displaces the
// interface currently being displayed. Without hysteresis the display flips
// between two interfaces whenever their traffic is similar, which reads as the
// numbers randomly changing.
const switchRatio = 3

// switchFloor is the rate below which the current interface is considered idle.
// A challenger must clear this absolute floor as well as switchRatio, so idle
// background traffic on a quiet link never displaces a busy one.
const switchFloor = 8 << 10 // 8 KiB/s

func New(iface string) *Collector {
	if iface == "" {
		iface = "auto"
	}
	return &Collector{
		iface: iface,
		prev:  make(map[string]counter),
	}
}

func (c *Collector) Read() (Snapshot, error) {
	stats, err := gnet.IOCounters(true)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collector: read interface counters: %w", err)
	}
	if len(stats) == 0 {
		return Snapshot{}, fmt.Errorf("collector: no network interfaces found")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.iface != "auto" {
		for _, s := range stats {
			if s.Name == c.iface {
				c.prev[s.Name] = counter{bytes: s.BytesRecv + s.BytesSent, at: time.Now()}
				return Snapshot{Interface: s.Name, RxBytes: s.BytesRecv, TxBytes: s.BytesSent}, nil
			}
		}
		return Snapshot{}, fmt.Errorf("collector: interface %q not found%s",
			c.iface, availableList(stats))
	}

	name, rx, tx := c.pickAuto(stats)
	if name == "" {
		return Snapshot{}, fmt.Errorf("collector: no usable network interface found (only loopback and virtual interfaces are up)")
	}
	return Snapshot{Interface: name, RxBytes: rx, TxBytes: tx}, nil
}

// pickAuto chooses the interface currently carrying the most traffic.
//
// Selection is by recent rate, not by lifetime totals. Ranking by totals looks
// reasonable until a machine has been up for a while: a long-lived NIC that has
// transferred a lot over its lifetime wins even while it sits idle, so a
// throughput test on another interface looks like ~10 KB/s of background noise.
func (c *Collector) pickAuto(stats []gnet.IOCountersStat) (string, uint64, uint64) {
	now := time.Now()
	elapsed := now.Sub(c.lastAt).Seconds()
	if c.lastAt.IsZero() || elapsed <= 0 {
		elapsed = 0
	}
	c.lastAt = now

	// Pick a different interface to the current one; only these can win.
	candidates := make([]gnet.IOCountersStat, 0, len(stats))
	for i := range stats {
		if isVirtual(stats[i].Name) {
			continue
		}
		candidates = append(candidates, stats[i])
	}
	if len(candidates) == 0 {
		return "", 0, 0
	}

	// Capture the previous reading for every candidate BEFORE recording this
	// one, otherwise every delta would be zero.
	previous := make(map[string]uint64, len(candidates))
	for i := range candidates {
		s := candidates[i]
		if p, ok := c.prev[s.Name]; ok {
			previous[s.Name] = p.bytes
		}
	}

	// Remember this reading for the next rate calculation, and drop interfaces
	// that disappeared so their counters cannot produce a bogus delta.
	seen := make(map[string]bool, len(candidates))
	for i := range candidates {
		s := candidates[i]
		seen[s.Name] = true
		c.prev[s.Name] = counter{bytes: s.BytesRecv + s.BytesSent, at: now}
	}
	for name := range c.prev {
		if !seen[name] {
			delete(c.prev, name)
		}
	}

	// Fall back to lifetime totals on the first read: there is no interval yet,
	// so no rate exists.
	if elapsed == 0 {
		best := pickBest(candidates)
		if best == nil {
			return "", 0, 0
		}
		// Record the incumbent so the next read applies hysteresis against it.
		c.current = best.Name
		return best.Name, best.BytesRecv, best.BytesSent
	}

	rates := make(map[string]float64, len(candidates))
	curRate := 0.0
	for i := range candidates {
		s := candidates[i]
		prevBytes, ok := previous[s.Name]
		if !ok {
			continue // first sighting of this interface
		}
		total := s.BytesRecv + s.BytesSent
		// A decrease means the counter reset (interface restart, driver reload),
		// not negative traffic: report nothing for this interval.
		if total < prevBytes {
			continue
		}
		rates[s.Name] = float64(total-prevBytes) / elapsed
		if s.Name == c.current {
			curRate = rates[s.Name]
		}
	}

	// Keep the current interface unless a challenger is clearly busier.
	if c.current != "" {
		if _, ok := rates[c.current]; ok {
			bestChallenger, bestRate := "", 0.0
			for name, r := range rates {
				if name == c.current {
					continue
				}
				if r > bestRate {
					bestChallenger, bestRate = name, r
				}
			}
			if bestChallenger == "" {
				// Nothing to switch to; the incumbent holds.
				return countersFor(candidates, c.current)
			}
			if bestRate >= switchFloor && bestRate > curRate*switchRatio {
				// Report the challenger immediately. Falling through here would
				// re-resolve by lifetime totals and undo the switch, which is
				// exactly the bug this selection policy exists to fix.
				c.current = bestChallenger
				return countersFor(candidates, bestChallenger)
			}
			// Incumbent holds.
			return countersFor(candidates, c.current)
		}
	}

	best := pickBest(candidates)
	if best == nil {
		return "", 0, 0
	}
	c.current = best.Name
	return best.Name, best.BytesRecv, best.BytesSent
}

// countersFor returns the counters of the named interface among candidates.
func countersFor(candidates []gnet.IOCountersStat, name string) (string, uint64, uint64) {
	for i := range candidates {
		if candidates[i].Name == name {
			return name, candidates[i].BytesRecv, candidates[i].BytesSent
		}
	}
	return "", 0, 0
}

// availableList renders the usable interfaces for an error message, so a typo
// in --interface is immediately correctable.
func availableList(stats []gnet.IOCountersStat) string {
	names := make([]string, 0, len(stats))
	for _, s := range stats {
		names = append(names, s.Name)
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return "\n\navailable interfaces:\n  " + strings.Join(names, "\n  ")
}

func Interfaces() ([]string, error) {
	stats, err := gnet.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("collector: read interface counters: %w", err)
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
		return nil, fmt.Errorf("collector: list interfaces: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Name != name {
			continue
		}
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
	return nil, fmt.Errorf("collector: interface %q not found%s", name, availableNames(interfaces))
}

func availableNames(interfaces []gnet.InterfaceStat) string {
	if len(interfaces) == 0 {
		return ""
	}
	names := make([]string, 0, len(interfaces))
	for _, i := range interfaces {
		names = append(names, i.Name)
	}
	sort.Strings(names)
	return "\n\navailable interfaces:\n  " + strings.Join(names, "\n  ")
}

// pickBest ranks by lifetime totals. It is the tie-break for the first read,
// before any rate can be derived, and the fallback when nothing is moving.
func pickBest(stats []gnet.IOCountersStat) *gnet.IOCountersStat {
	var best *gnet.IOCountersStat
	var bestTotal uint64
	for i := range stats {
		s := &stats[i]
		if isVirtual(s.Name) {
			continue
		}
		total := s.BytesRecv + s.BytesSent
		if best == nil || total > bestTotal {
			best, bestTotal = s, total
		}
	}
	return best
}

// isVirtual reports whether an interface should be excluded from automatic
// selection: loopback and the virtual bridges that containers and VMs create.
func isVirtual(name string) bool {
	switch name {
	case "lo", "lo0":
		return true
	}
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

var virtualPrefixes = []string{"docker", "br-", "veth", "virbr", "vmnet", "vbox", "tun", "tap", "wg", "zt"}

// pickAutoForTest drives auto-selection with a supplied counters set and
// returns the chosen interface. It exists so the selection policy can be tested
// against specific traffic shapes without depending on real host interfaces.
func (c *Collector) pickAutoForTest(s []gnet.IOCountersStat) string {
	name, _, _ := c.pickAuto(s)
	return name
}
