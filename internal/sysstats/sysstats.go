// Package sysstats reports CPU, memory, battery and uptime on macOS and Linux.
package sysstats

import (
	"context"
	"math"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v3/mem"
)

// Service samples system metrics with a controlled lifecycle.
type Service struct {
	mu       sync.RWMutex
	cpuPct   float64
	hostname string        // Cached to avoid constant system calls
	stopChan chan struct{} // Controls the background goroutine shutdown
}

// commandOutput runs a command and returns stdout. Indirected so tests can
// simulate command failures.
var commandOutput = func(c *exec.Cmd) ([]byte, error) { return c.Output() }

// New starts the background CPU sampler and returns the Service
func New() *Service {
	name, _ := os.Hostname() // Fetch once at startup
	s := &Service{
		hostname: name,
		stopChan: make(chan struct{}), // kill switch for goroutine to avoid leaks
	}
	go s.sampleLoop()
	return s
}

// Close terminates the background monitoring goroutine to avoid goroutine leak
func (s *Service) Close() {
	close(s.stopChan)
}

func (s *Service) sampleLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return // prevents goroutine leaking on service destruction
		case <-ticker.C:
			if p, ok := sampleCPUPercent(); ok {
				s.mu.Lock()
				s.cpuPct = round1(p)
				s.mu.Unlock()
			}
		}
	}
}

// Get returns a fresh snapshot.
func (s *Service) Get(ctx context.Context) (Stats, error) {
	var st Stats

	// Abort immediately if the frontend cancelled the request
	if err := ctx.Err(); err != nil {
		return st, err
	}

	s.mu.RLock()
	st.CPUPercent = s.cpuPct
	st.Hostname = s.hostname // ◄ Fast memory read instead of Syscall
	s.mu.RUnlock()

	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		st.MemPercent = round1(vm.UsedPercent)
		st.MemUsedGB = round1(float64(vm.Used) / (1 << 30))
		st.MemTotalGB = round1(float64(vm.Total) / (1 << 30))
	}

	// Double-check context before hitting the disk/battery controllers
	if err := ctx.Err(); err != nil {
		return st, err
	}

	st.UptimeHours = round1(float64(uptimeSeconds()) / 3600)
	st.BatteryPct, st.BatteryState = readBattery()
	return st, nil
}

// readBattery only works on laptops, for PC - it will simply be ignored
func readBattery() (float64, string) {
	bats, err := battery.GetAll()
	if err != nil || len(bats) == 0 {
		return -1, "n/a"
	}
	b := bats[0]
	pct := 0.0
	if b.Full > 0 {
		pct = round1(b.Current / b.Full * 100)
	}
	switch b.State.String() {
	case "Charging":
		return pct, "charging"
	case "Full":
		return pct, "full"
	case "Empty", "Discharging":
		return pct, "discharging"
	default:
		return pct, "discharging"
	}
}

// Fixed to use deterministic, cross-platform standard math
func round1(f float64) float64 {
	return math.Round(f*10) / 10
}
