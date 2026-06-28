//go:build !windows

// package sysstats provides system statistics for the Apollo widget,
// including CPU and memory usage, and top processes. This file contains Unix-specific implementations.
// build script and cross platform support lead to separate OS level info gathering.
// This file is for Unix (shared for MacOS and Linux), other OSes have their own implementations.
package sysstats

import (
	"context"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// TopProcesses (AI supported logic) returns the 5 processes using the most CPU. It shells out to
// `ps` (present on both macOS and Linux) to avoid the cgo process libraries.
func (s *Service) TopProcesses(ctx context.Context) ([]Process, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// -r sorts by current CPU usage; trailing "=" suppresses headers.
		cmd = exec.CommandContext(ctx, "ps", "-axo", "pcpu=,pmem=,comm=", "-r")
	default: // linux and friends
		cmd = exec.CommandContext(ctx, "ps", "-eo", "pcpu=,pmem=,comm=", "--sort=-pcpu")
	}

	out, err := commandOutput(cmd)
	if err != nil {
		return nil, err
	}

	// Aggregate by app name so an app's many helper processes (Chrome, VS Code, …) collapse into a single row summing their CPU/memory.
	byName := map[string]*Process{}
	var order []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		cpu, err1 := strconv.ParseFloat(fields[0], 64)
		mem, err2 := strconv.ParseFloat(fields[1], 64)
		if err1 != nil || err2 != nil {
			continue
		}
		name := cleanName(strings.Join(fields[2:], " "))
		if name == "" {
			continue
		}
		if p, ok := byName[name]; ok {
			p.CPU += cpu
			p.Mem += mem
		} else {
			byName[name] = &Process{Name: name, CPU: cpu, Mem: mem}
			order = append(order, name)
		}
	}

	procs := make([]Process, 0, len(order))
	for _, name := range order {
		p := byName[name]
		procs = append(procs, Process{Name: p.Name, CPU: round1(p.CPU), Mem: round1(p.Mem)})
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].CPU > procs[j].CPU })
	if len(procs) > 5 {
		procs = procs[:5]
	}
	return procs, nil
}
