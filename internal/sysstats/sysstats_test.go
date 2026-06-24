package sysstats

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestServiceLifecycleAndSampleLoop(t *testing.T) {
	// Create service, which instantly spins up the sampleLoop goroutine
	s := New()

	// Wait briefly to allow at least one loop or tick registration if needed,
	// then call Close to execute the s.stopChan channel selection.
	time.Sleep(10 * time.Millisecond)
	s.Close()
}

func TestGetContextCancelled(t *testing.T) {
	s := New()
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel instantly before execution

	// Triggers the first ctx.Err() short-circuit check
	_, err := s.Get(ctx)
	if err == nil {
		t.Error("expected error when context is cancelled at entry block")
	}
}

func TestTopProcessesCommandFailure(t *testing.T) {
	orig := commandOutput
	commandOutput = func(*exec.Cmd) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { commandOutput = orig }()

	s := New()
	defer s.Close()

	if _, err := s.TopProcesses(context.Background()); err == nil {
		t.Error("TopProcesses should error when command fails")
	}
}

func TestWindowsTopProcessesParsing(t *testing.T) {
	orig := commandOutput
	defer func() { commandOutput = orig }()

	s := New()
	defer s.Close()

	t.Run("single json object response", func(t *testing.T) {
		commandOutput = func(*exec.Cmd) ([]byte, error) {
			return []byte(`{"Name":"chrome","CPU":12.5,"Mem":104857600}`), nil
		}
		procs, err := s.TopProcesses(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(procs) != 1 || procs[0].Name != "chrome" || procs[0].Mem != 100.0 {
			t.Errorf("unexpected structure output parsing single object: %+v", procs)
		}
	})

	t.Run("array json response", func(t *testing.T) {
		commandOutput = func(*exec.Cmd) ([]byte, error) {
			return []byte(`[{"Name":"pwsh","CPU":5.0,"Mem":52428800}]`), nil
		}
		procs, err := s.TopProcesses(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(procs) != 1 || procs[0].Name != "pwsh" || procs[0].Mem != 50.0 {
			t.Errorf("unexpected structure output parsing array: %+v", procs)
		}
	})
}

func TestWindowsCleanName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"explorer.exe", "explorer"},
		{"TASKMGR.EXE", "TASKMGR"},
		{"sans-extension", "sans-extension"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := cleanName(tt.in); got != tt.want {
				t.Errorf("cleanName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
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

func TestGet(t *testing.T) {
	s := New()
	defer s.Close()

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
	defer s.Close()

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

func numCPUGuard() int { return 64 }

func TestUptimeSeconds(t *testing.T) {
	if uptimeSeconds() < 0 {
		t.Error("uptime negative")
	}
}
