//go:build linux

package sysstats

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// sampleCPUPercent (AI supported logic) reads /proc/stat twice and computes the busy fraction over a
// short window. No cgo.
func sampleCPUPercent() (float64, bool) {
	idle1, total1, ok1 := readProcStat()
	if !ok1 {
		return 0, false
	}
	time.Sleep(250 * time.Millisecond)
	idle2, total2, ok2 := readProcStat()
	if !ok2 {
		return 0, false
	}
	dTotal := float64(total2 - total1)
	dIdle := float64(idle2 - idle1)
	if dTotal <= 0 {
		return 0, false
	}
	return (1 - dIdle/dTotal) * 100, true
}

// readProcStat returns aggregate idle and total jiffies from the "cpu" line.
func readProcStat() (idle, total uint64, ok bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		for i, f := range fields {
			v, err := strconv.ParseUint(f, 10, 64)
			if err != nil {
				continue
			}
			total += v
			// fields: user nice system idle iowait irq softirq steal ...
			if i == 3 || i == 4 { // idle + iowait
				idle += v
			}
		}
		return idle, total, true
	}
	return 0, 0, false
}

// uptimeSeconds reads /proc/uptime.
func uptimeSeconds() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	secs, _ := strconv.ParseFloat(fields[0], 64)
	return int64(secs)
}
