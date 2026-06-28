// Package sysstats reports CPU, memory, battery and uptime on macOS and Linux.
package sysstats

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v3/mem"
)

// Service samples system metrics with a controlled lifecycle.
type Service struct {
	mu        sync.RWMutex
	cpuPct    float64
	hostname  string        // Cached to avoid constant system calls
	stopChan  chan struct{} // Controls the background goroutine shutdown
	closeOnce sync.Once     // unit test panic caused this, to avoid double closure of the stopChan
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
// closeOnce ensures that the stopChan is closed only once,
// preventing potential panics from multiple close calls.
func (s *Service) Close() {
	s.closeOnce.Do(func() {
		close(s.stopChan)
	})
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

// Expose a package-level variable for battery retrieval so we can swap it out in tests
var getBatteryInfo = func() ([]*battery.Battery, error) {
	return battery.GetAll()
}

func readBattery() (float64, string) {
	// Call through our variable hook instead of calling the package directly
	bats, err := getBatteryInfo()
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

func cleanName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// 1. Convert backslashes to forward slashes so logic works on all OSs
	path = strings.ReplaceAll(path, "\\", "/")

	// 2. Handle macOS .app bundles before getting the Base name
	// This works for ".../Foo.app/Contents/MacOS/Foo" -> ".../Foo.app"
	if i := strings.Index(path, ".app/"); i >= 0 {
		path = path[:i+4]
	}

	// 3. Get the last element (e.g., "chrome.exe" or "Foo.app")
	path = filepath.Base(path)

	// 4. Strip common extensions
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".exe") {
		path = path[:len(path)-4]
	}
	if strings.HasSuffix(lower, ".app") {
		path = path[:len(path)-4]
	}

	return path
}

// Fixed to use deterministic, cross-platform standard math
func round1(f float64) float64 {
	return math.Round(f*10) / 10
}
