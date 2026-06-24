//go:build darwin

package sysstats

import (
	"errors"
	"os/exec"
	"testing"
)

func TestDarwinCommandFailures(t *testing.T) {
	orig := commandOutput
	commandOutput = func(*exec.Cmd) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { commandOutput = orig }()

	if _, ok := sampleCPUPercent(); ok {
		t.Error("sampleCPUPercent should fail when command fails")
	}
	if uptimeSeconds() != 0 {
		t.Error("uptimeSeconds should be 0 when command fails")
	}

	// Command succeeds but output is unparseable → boot==0 branch.
	commandOutput = func(*exec.Cmd) ([]byte, error) { return []byte("no boottime here"), nil }
	if uptimeSeconds() != 0 {
		t.Error("uptimeSeconds should be 0 when boottime unparseable")
	}
}
