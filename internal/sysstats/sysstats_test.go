package sysstats

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/distatus/battery"
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
		{"macos bundle", "/Applications/test.app/Contents/MacOS/test", "test"},
		{"windows path", `C:\Program Files\Google\Chrome.exe`, "Chrome"},
		{"linux path", "/usr/bin/python3", "python3"},
		{"bare name", "WindowServer", "WindowServer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanName(tt.in); got != tt.want {
				t.Errorf("cleanName(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTopProcessesCommandFailure(t *testing.T) {
	orig := commandOutput
	commandOutput = func(*exec.Cmd) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { commandOutput = orig }()

	s := New()
	defer s.Close()

	// This will now correctly trigger the error on Windows AND Unix
	// because they share the same commandOutput variable.
	if _, err := s.TopProcesses(context.Background()); err == nil {
		t.Error("TopProcesses should error when command fails")
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

func makeBat(state battery.AgnosticState, current, full float64) *battery.Battery {
	b := &battery.Battery{Current: current, Full: full}
	b.State.Raw = state
	return b
}

func TestReadBattery(t *testing.T) {
	orig := getBatteryInfo
	defer func() { getBatteryInfo = orig }()

	tests := []struct {
		name      string
		bats      []*battery.Battery
		err       error
		wantPct   float64
		wantState string
	}{
		{"error → n/a", nil, errors.New("no battery"), -1, "n/a"},
		{"empty slice → n/a", []*battery.Battery{}, nil, -1, "n/a"},
		{"charging", []*battery.Battery{makeBat(battery.Charging, 60, 100)}, nil, 60, "charging"},
		{"full", []*battery.Battery{makeBat(battery.Full, 100, 100)}, nil, 100, "full"},
		{"discharging", []*battery.Battery{makeBat(battery.Discharging, 50, 100)}, nil, 50, "discharging"},
		{"empty state", []*battery.Battery{makeBat(battery.Empty, 0, 100)}, nil, 0, "discharging"},
		{"unknown state falls back", []*battery.Battery{makeBat(battery.Unknown, 80, 100)}, nil, 80, "discharging"},
		{"zero full → 0%", []*battery.Battery{makeBat(battery.Charging, 50, 0)}, nil, 0, "charging"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bats, e := tt.bats, tt.err
			getBatteryInfo = func() ([]*battery.Battery, error) { return bats, e }
			pct, state := readBattery()
			if pct != tt.wantPct {
				t.Errorf("pct=%v want %v", pct, tt.wantPct)
			}
			if state != tt.wantState {
				t.Errorf("state=%q want %q", state, tt.wantState)
			}
		})
	}
}
