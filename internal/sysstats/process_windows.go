//go:build windows

package sysstats

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// WindowsProcess maps the output string from our PowerShell pipeline
type WindowsProcess struct {
	Name string  `json:"Name"`
	CPU  float64 `json:"CPU"`
	Mem  float64 `json:"Mem"`
}

// TopProcesses (AI supported logic) executes a lightweight PowerShell script to aggregate and extract
// process stats without CGO.
func (s *Service) TopProcesses(ctx context.Context) ([]Process, error) {
	// This script grabs processes, groups them by name to combine sub-tasks,
	// calculates total CPU, and formats it directly into JSON.
	psCmd := `Get-Process | Where-Object {$_.CPU -gt 0} | ` +
		`Group-Object Name | ForEach-Object { ` +
		`[PSCustomObject]@{ Name = $_.Name; CPU = ($_.Group | Measure-Object CPU -Sum).Sum; Mem = ($_.Group | Measure-Object WorkingSet -Sum).Sum } ` +
		`} | Sort-Object CPU -Descending | Select-Object -First 5 | ConvertTo-Json`

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var rawProcs []WindowsProcess
	// Handle single object vs array JSON edge-cases from PowerShell
	if len(out) > 0 && out[0] == '{' {
		var single WindowsProcess
		if err := json.Unmarshal(out, &single); err == nil {
			rawProcs = append(rawProcs, single)
		}
	} else {
		_ = json.Unmarshal(out, &rawProcs)
	}

	procs := make([]Process, 0, len(rawProcs))
	for _, rp := range rawProcs {
		// Convert WorkingSet bytes to a human-readable Megabyte value
		memMB := rp.Mem / (1024 * 1024)
		procs = append(procs, Process{
			Name: rp.Name,
			CPU:  round1(rp.CPU),
			Mem:  round1(memMB),
		})
	}

	return procs, nil
}

// added for cross-platform testing issue resolution. not really needed for windows
func cleanName(name string) string {
	// Example Windows-specific cleaning: strip out trailing .exe case-insensitively
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".exe") {
		return name[:len(name)-4]
	}
	return name
}
