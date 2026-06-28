//go:build windows

package sysstats

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

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

	t.Run("command failure", func(t *testing.T) {
		commandOutput = func(*exec.Cmd) ([]byte, error) { return nil, errors.New("boom") }
		if _, err := s.TopProcesses(context.Background()); err == nil {
			t.Error("TopProcesses should error when command fails")
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
