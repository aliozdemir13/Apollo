//go:build linux

package sysstats

import (
	"os"
	"testing"
)

func TestReadProcStat(t *testing.T) {
	// Mock /proc/stat content
	// fields: user nice system idle iowait irq softirq steal
	mockContent := []byte("cpu  100 100 100 1000 100 0 0 0\n")

	// Override the readFile function
	oldReadFile := readFile
	readFile = func(name string) ([]byte, error) {
		return mockContent, nil
	}
	defer func() { readFile = oldReadFile }()

	idle, total, ok := readProcStat()
	if !ok {
		t.Fatal("Expected readProcStat to succeed")
	}

	// Calculations:
	// Total = 100+100+100+1000+100 = 1400
	// Idle = Index 3 (1000) + Index 4 (100) = 1100
	if total != 1400 {
		t.Errorf("Expected total 1400, got %d", total)
	}
	if idle != 1100 {
		t.Errorf("Expected idle 1100, got %d", idle)
	}
}

func TestUptimeSecondsLinux(t *testing.T) {
	mockContent := []byte("1234.56 7890.12\n")

	oldReadFile := readFile
	readFile = func(name string) ([]byte, error) {
		return mockContent, nil
	}
	defer func() { readFile = oldReadFile }()

	uptime := uptimeSeconds()
	if uptime != 1234 {
		t.Errorf("Expected uptime 1234, got %d", uptime)
	}
}

func TestSampleCPUPercentLinux(t *testing.T) {
	// We want to simulate a change in CPU usage.
	// Call 1: Total 1000, Idle 900 (10% busy)
	// Call 2: Total 2000, Idle 1400 (60% busy)
	// Delta Total: 1000, Delta Idle: 500
	// Busy = (1 - 500/1000) * 100 = 50%

	callCount := 0
	oldReadFile := readFile
	readFile = func(name string) ([]byte, error) {
		callCount++
		if callCount == 1 {
			return []byte("cpu  50 0 50 900 0 0 0 0\n"), nil
		}
		return []byte("cpu  350 0 250 1400 0 0 0 0\n"), nil
	}
	defer func() { readFile = oldReadFile }()

	pct, ok := sampleCPUPercent()
	if !ok {
		t.Fatal("Expected sampleCPUPercent to succeed")
	}

	if pct != 50.0 {
		t.Errorf("Expected 50%% CPU usage, got %f%%", pct)
	}
}

func TestReadProcStat_InvalidFile(t *testing.T) {
	oldReadFile := readFile
	readFile = func(name string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	defer func() { readFile = oldReadFile }()

	_, _, ok := readProcStat()
	if ok {
		t.Error("Expected failure for non-existent file")
	}
}
