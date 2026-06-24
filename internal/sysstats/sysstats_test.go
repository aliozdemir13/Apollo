package sysstats

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestTopProcessesCommandFailure(t *testing.T) {
	orig := commandOutput
	commandOutput = func(*exec.Cmd) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { commandOutput = orig }()

	if _, err := New().TopProcesses(context.Background()); err == nil {
		t.Error("TopProcesses should error when command fails")
	}
}

func TestRound1(t *testing.T) {
	tests := []struct {
		in   float64
		want float64
	}{
		{0, 0},
		{1.24, 1.2},
		{1.25, 1.3},
		{1.299, 1.3},
		{99.95, 100},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := round1(tt.in); got != tt.want {
				t.Errorf("round1(%v)=%v want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCleanName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"macos app bundle", "/Applications/Claude.app/Contents/MacOS/Claude Helper", "Claude"},
		{"plain path", "/usr/sbin/bluetoothd", "bluetoothd"},
		{"bare name", "WindowServer", "WindowServer"},
		{"empty", "", ""},
		{"whitespace", "   ", ""},
		{"linux comm", "/usr/lib/firefox/firefox", "firefox"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanName(tt.in); got != tt.want {
				t.Errorf("cleanName(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestGet runs the real collectors (mem/battery/uptime/hostname) available on
// the host and asserts sane values.
func TestGet(t *testing.T) {
	s := New()
	st, err := s.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.MemTotalGB <= 0 || st.MemPercent < 0 || st.MemPercent > 100 {
		t.Errorf("mem looks wrong: %+v", st)
	}
	if st.Hostname == "" {
		t.Error("hostname empty")
	}
	if st.BatteryPct < -1 || st.BatteryPct > 100 {
		t.Errorf("battery pct out of range: %v", st.BatteryPct)
	}
	switch st.BatteryState {
	case "charging", "discharging", "full", "n/a":
	default:
		t.Errorf("unexpected battery state %q", st.BatteryState)
	}
}

func TestTopProcesses(t *testing.T) {
	s := New()
	procs, err := s.TopProcesses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(procs) > 5 {
		t.Errorf("expected at most 5, got %d", len(procs))
	}
	for i := 1; i < len(procs); i++ {
		if procs[i-1].CPU < procs[i].CPU {
			t.Errorf("not sorted desc by cpu: %+v", procs)
		}
	}
}

func TestSampleCPUPercent(t *testing.T) {
	pct, ok := sampleCPUPercent()
	if ok && (pct < 0 || pct > 100*float64(numCPUGuard())) {
		t.Errorf("cpu%% out of range: %v", pct)
	}
}

// numCPUGuard keeps the CPU upper bound generous (aggregate can exceed 100).
func numCPUGuard() int { return 64 }

func TestUptimeSeconds(t *testing.T) {
	if uptimeSeconds() < 0 {
		t.Error("uptime negative")
	}
}
