//go:build darwin

package sysstats

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// sampleCPUPercent (AI supported logic) shells out to `top`, which reports a recent CPU-usage sample
// without needing cgo (and so avoids the go-m1cpu crash). `-l 2` takes two
// samples; the second reflects activity over the sampling interval.
func sampleCPUPercent() (float64, bool) {
	out, err := exec.Command("top", "-l", "2", "-n", "0", "-s", "1").Output()
	if err != nil {
		return 0, false
	}
	lines := strings.Split(string(out), "\n")
	idle := -1.0
	for _, line := range lines {
		// "CPU usage: 4.76% user, 9.52% sys, 85.71% idle"
		i := strings.Index(line, "CPU usage:")
		if i < 0 {
			continue
		}
		for _, part := range strings.Split(line[i:], ",") {
			part = strings.TrimSpace(part)
			if strings.HasSuffix(part, "idle") {
				fields := strings.Fields(part) // ["85.71%", "idle"]
				if len(fields) >= 1 {
					v, err := strconv.ParseFloat(strings.TrimSuffix(fields[0], "%"), 64)
					if err == nil {
						idle = v // keep the last (most recent) sample
					}
				}
			}
		}
	}
	if idle < 0 {
		return 0, false
	}
	return 100 - idle, true
}

// uptimeSeconds parses `sysctl -n kern.boottime`, e.g.
// "{ sec = 1718200000, usec = 0 } Tue Jun 10 12:00:00 2025".
func uptimeSeconds() int64 {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return 0
	}
	s := string(out)
	i := strings.Index(s, "sec =")
	if i < 0 {
		return 0
	}
	rest := s[i+len("sec ="):]
	rest = strings.TrimSpace(rest)
	end := strings.IndexAny(rest, ", }")
	if end > 0 {
		rest = rest[:end]
	}
	boot, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
	if err != nil || boot == 0 {
		return 0
	}
	return time.Now().Unix() - boot
}
