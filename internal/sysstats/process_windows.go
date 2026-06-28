//go:build windows

// package sysstats provides system statistics for the Apollo widget,
// including CPU and memory usage, and top processes. This file contains Windows-specific implementations.
// build script and cross platform support lead to separate OS level info gathering.
// This file is for Windows, other OSes have their own implementations.
package sysstats

import (
	"context"
	"sort"

	"github.com/shirou/gopsutil/v3/process"
)

func (s *Service) TopProcesses(ctx context.Context) ([]Process, error) {
	// 1. Get all running processes
	allProcs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	type procSnap struct {
		Name string
		CPU  float64
		Mem  float64
	}
	var snaps []procSnap

	for _, p := range allProcs {
		// Get CPU percent.
		// Note: The first time this is called on a process object, it returns 0.
		// For a widget, you might want to cache process objects to get delta-based percentages,
		// but Percent(0) usually works for a "snapshot" of current activity.
		cpu, err := p.CPUPercentWithContext(ctx)
		if err != nil || cpu <= 0 {
			continue
		}

		name, err := p.NameWithContext(ctx)
		if err != nil {
			continue
		}

		memInfo, err := p.MemoryInfoWithContext(ctx)
		if err != nil {
			continue
		}

		snaps = append(snaps, procSnap{
			Name: name,
			CPU:  cpu,
			Mem:  float64(memInfo.RSS) / (1024 * 1024), // Convert to MB
		})
	}

	// 2. Sort by CPU descending
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].CPU > snaps[j].CPU
	})

	// 3. Take Top 5 and format
	limit := 5
	if len(snaps) < 5 {
		limit = len(snaps)
	}

	procs := make([]Process, 0, limit)
	for i := 0; i < limit; i++ {
		procs = append(procs, Process{
			Name: cleanName(snaps[i].Name),
			CPU:  round1(snaps[i].CPU),
			Mem:  round1(snaps[i].Mem),
		})
	}

	return procs, nil
}
