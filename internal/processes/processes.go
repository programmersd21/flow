// internal/processes/processes.go
//
// Lists active network processes and their connection counts.
// Uses gopsutil to enumerate TCP connections and resolve process info.
//
// Per-process bandwidth accounting is not available without elevated
// privileges or platform-specific APIs (e.g. eBPF, netlink). This
// package falls back to connection-count view, which works on all
// platforms without overhead.

package processes

import (
	"fmt"
	"sort"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type Info struct {
	PID         int
	Name        string
	Connections int
}

// List enumerates TCP endpoints (established connections and listeners),
// groups them by PID, resolves process names, and returns them sorted by
// connection count descending.
func List() ([]Info, error) {
	conns, err := net.Connections("tcp")
	if err != nil {
		return nil, fmt.Errorf("processes: connections: %w", err)
	}

	pidCount := make(map[int]int)
	for _, c := range conns {
		if c.Pid > 0 {
			pidCount[int(c.Pid)]++
		}
	}

	if len(pidCount) == 0 {
		return nil, nil
	}

	result := make([]Info, 0, len(pidCount))
	for pid, count := range pidCount {
		name := resolveName(pid)
		result = append(result, Info{
			PID:         pid,
			Name:        name,
			Connections: count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Connections > result[j].Connections
	})

	return result, nil
}

func resolveName(pid int) string {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return "?"
	}
	name, err := p.Name()
	if err != nil || name == "" {
		return "?"
	}
	return name
}
