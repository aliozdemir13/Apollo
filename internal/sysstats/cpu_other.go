//go:build !linux && !darwin && !windows

// package sysstats provides system statistics for the Apollo widget,
// fallback script.
package sysstats

// Fallback for platforms we don't specifically support.
func sampleCPUPercent() (float64, bool) { return 0, false }

func uptimeSeconds() int64 { return 0 }
