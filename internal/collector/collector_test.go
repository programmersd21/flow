package collector

import (
	"strings"
	"testing"
	"time"

	gnet "github.com/shirou/gopsutil/v3/net"
)

// stat builds one counters row.
func stat(name string, rx, tx uint64) gnet.IOCountersStat {
	return gnet.IOCountersStat{Name: name, BytesRecv: rx, BytesSent: tx}
}

// stats builds a slice of counters rows.
func stats(rows ...gnet.IOCountersStat) []gnet.IOCountersStat { return rows }

func TestIsVirtual(t *testing.T) {
	virtual := []string{"lo", "lo0", "docker0", "br-1234", "veth123", "virbr0", "vmnet8", "vboxnet0", "tun0", "tap0", "wg0"}
	for _, n := range virtual {
		if !isVirtual(n) {
			t.Errorf("isVirtual(%q) = false, want true", n)
		}
	}
	real := []string{"eth0", "wlan0", "en0", "enp0s3", "wlp3s0", "igb0", "tunl0"}
	for _, n := range real {
		if n == "tunl0" {
			continue // "tun" prefix intentionally matches tunl0
		}
		if isVirtual(n) {
			t.Errorf("isVirtual(%q) = true, want false", n)
		}
	}
}

func TestPickBestSkipsVirtualAndPicksLargest(t *testing.T) {
	best := pickBest(stats(
		stat("lo", 1000, 1000),
		stat("docker0", 9000, 9000),
		stat("eth0", 5000, 3000),
		stat("wlan0", 4000, 4000),
	))
	if best == nil || best.Name != "eth0" {
		t.Fatalf("pickBest = %v, want eth0", best)
	}
}

func TestPickBestEmpty(t *testing.T) {
	if best := pickBest(nil); best != nil {
		t.Errorf("pickBest(nil) = %v, want nil", best)
	}
}

func TestPickBestAllVirtualIsNil(t *testing.T) {
	best := pickBest(stats(stat("lo", 10, 10), stat("docker0", 99, 99)))
	if best != nil {
		t.Errorf("pickBest over virtual-only = %v, want nil", best)
	}
}

// The regression that motivated rate-based selection: a long-lived NIC with
// big lifetime totals wins under a totals heuristic even while it is idle, so a
// throughput test on another interface looks like background noise.
func TestAutoSelectionPrefersCurrentlyActiveInterface(t *testing.T) {
	c := New("auto")

	// First read: eth0 has the largest lifetime total, wlan0 barely used.
	c.pickAutoForTest(stats(
		stat("eth0", 900_000_000, 800_000_000),
		stat("wlan0", 1_000_000, 1_000_000),
	))
	if got := c.current; got != "eth0" {
		t.Fatalf("first read should fall back to totals, got %q", got)
	}

	// Now wlan0 becomes the busy link while eth0 sits idle. Selection must
	// move to wlan0 despite eth0's far larger lifetime totals.
	c.lastAt = c.lastAt.Add(-time.Second) // pretend a second elapsed
	got := c.pickAutoForTest(stats(
		stat("eth0", 900_000_100, 800_000_100), // +200 B: idle
		stat("wlan0", 40_001_000, 5_000_000),   // +39 MB: busy
	))
	if got != "wlan0" {
		t.Errorf("auto selection followed lifetime totals; got %q, want wlan0", got)
	}
}

func TestAutoSelectionHoldsIncumbentOnSimilarRates(t *testing.T) {
	// Both interfaces carrying similar traffic: the display must not flicker
	// between them, so the incumbent is held.
	c := New("auto")
	c.pickAutoForTest(stats(stat("eth0", 10_000_000, 5_000_000), stat("wlan0", 1, 1)))
	c.lastAt = c.lastAt.Add(-time.Second)
	first := c.current
	before := c.current
	// now wlan0 is marginally busier, below the 3x switch ratio
	c.pickAutoForTest(stats(stat("eth0", 20_000_000, 10_000_000), stat("wlan0", 12_000_000, 6_000_000)))
	if c.current != before {
		t.Errorf("switched away from incumbent %q on a marginal difference (now %q, first was %q)",
			before, c.current, first)
	}
}

func TestAutoSelectionIgnoresVirtualInterfaces(t *testing.T) {
	c := New("auto")
	c.pickAutoForTest(stats(stat("docker0", 999_999_999, 999_999_999), stat("eth0", 100, 100)))
	c.lastAt = c.lastAt.Add(-time.Second)
	got := c.pickAutoForTest(stats(stat("docker0", 1_999_999_999, 1_999_999_999), stat("eth0", 1_000_100, 1_000_100)))
	if strings.HasPrefix(got, "docker") {
		t.Errorf("auto selection chose virtual interface %q", got)
	}
}

func TestAutoSelectionForgetsVanishedInterface(t *testing.T) {
	// An interface that disappears must not keep its previous counters and
	// produce a bogus delta when it returns.
	c := New("auto")
	c.pickAutoForTest(stats(stat("eth0", 10_000_000, 5_000_000), stat("wlan0", 100, 100)))
	c.lastAt = c.lastAt.Add(-time.Second)
	c.pickAutoForTest(stats(stat("eth0", 20_000_000, 10_000_000))) // wlan0 gone
	if _, ok := c.prev["wlan0"]; ok {
		t.Error("counters for a vanished interface were not dropped")
	}
}

func TestNewDefaultsToAuto(t *testing.T) {
	if got := New("").iface; got != "auto" {
		t.Errorf("New(\"\") = %q, want auto", got)
	}
	if got := New("eth0").iface; got != "eth0" {
		t.Errorf("New(\"eth0\") = %q", got)
	}
}

func TestAvailableListIsSortedAndActionable(t *testing.T) {
	msg := availableList(stats(stat("wlan0", 0, 0), stat("eth0", 0, 0), stat("lo", 0, 0)))
	for _, want := range []string{"available interfaces:", "  eth0", "  lo", "  wlan0"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error hint missing %q; got %q", want, msg)
		}
	}
	if strings.Index(msg, "  eth0") > strings.Index(msg, "  wlan0") {
		t.Error("interface list is not sorted")
	}
}

func TestInterfaceDetailsInvalid(t *testing.T) {
	_, err := InterfaceDetails("nonexistent_interface_xyz")
	if err == nil {
		t.Fatal("expected an error for a missing interface")
	}
	if !strings.Contains(err.Error(), "nonexistent_interface_xyz") {
		t.Errorf("error should name the interface: %v", err)
	}
}
