package collector

import (
	"testing"

	gnet "github.com/shirou/gopsutil/v3/net"
)

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"lo", true},
		{"lo0", true},
		{"docker0", true},
		{"br-1234", true},
		{"veth123", true},
		{"virbr0", true},
		{"vmnet8", true},
		{"vboxnet0", true},
		{"eth0", false},
		{"wlan0", false},
		{"en0", false},
		{"enp0s3", false},
	}

	for _, tt := range tests {
		actual := isLoopback(tt.name)
		if actual != tt.expected {
			t.Errorf("isLoopback(%q) = %v; expected %v", tt.name, actual, tt.expected)
		}
	}
}

func TestPickBest_Empty(t *testing.T) {
	best := pickBest(nil)
	if best != nil {
		t.Errorf("pickBest(nil) = %v; expected nil", best)
	}
}

func TestPickBest_SkipsLoopback(t *testing.T) {
	stats := []gnet.IOCountersStat{
		{Name: "lo", BytesRecv: 1000, BytesSent: 1000},
		{Name: "eth0", BytesRecv: 5000, BytesSent: 3000},
	}
	best := pickBest(stats)
	if best == nil || best.Name != "eth0" {
		t.Errorf("pickBest should pick eth0, got %v", best)
	}
}

func TestNew(t *testing.T) {
	c := New("auto")
	if c == nil {
		t.Fatal("New('auto') returned nil")
	}
	if c.iface != "auto" {
		t.Errorf("New('auto').iface = %q; expected 'auto'", c.iface)
	}

	c2 := New("eth0")
	if c2.iface != "eth0" {
		t.Errorf("New('eth0').iface = %q; expected 'eth0'", c2.iface)
	}
}

func TestInterfaceDetails_Invalid(t *testing.T) {
	_, err := InterfaceDetails("nonexistent_interface_xyz")
	if err == nil {
		t.Error("InterfaceDetails for nonexistent interface should return error")
	}
}
