//go:build windows

// package sysstats provides system statistics for the Apollo widget,
// including CPU and memory usage, and top processes. This file contains Windows-specific implementations.
// build script and cross platform support lead to separate OS level info gathering.
// This file is for Windows, other OSes have their own implementations.
// Initially it is built via bash script, however command line quirks and unstable behaviour
// influenced the change of process.
package sysstats

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes = modkernel32.NewProc("GetSystemTimes")
	procGetTickCount64 = modkernel32.NewProc("GetTickCount64")

	getSystemTimesRaw = func() (idle, kernel, user uint64, ok bool) {
		var lpIdleTime, lpKernelTime, lpUserTime syscall.Filetime
		r1, _, _ := procGetSystemTimes.Call(
			uintptr(unsafe.Pointer(&lpIdleTime)),
			uintptr(unsafe.Pointer(&lpKernelTime)),
			uintptr(unsafe.Pointer(&lpUserTime)),
		)
		if r1 == 0 {
			return 0, 0, 0, false
		}
		return filetimeToUint64(lpIdleTime), filetimeToUint64(lpKernelTime), filetimeToUint64(lpUserTime), true
	}

	getTickCountRaw = func() uint64 {
		millis, _, _ := procGetTickCount64.Call()
		return uint64(millis)
	}
)

// sampleCPUPercent (AI supported logic) captures Windows system times twice and computes the busy
// fraction over a short window. No cgo.
func sampleCPUPercent() (float64, bool) {
	idle1, kernel1, user1, ok1 := getSystemTimesRaw()
	if !ok1 {
		return 0, false
	}

	time.Sleep(250 * time.Millisecond)

	idle2, kernel2, user2, ok2 := getSystemTimesRaw()
	if !ok2 {
		return 0, false
	}

	// On Windows, KernelTime returned by GetSystemTimes *includes* IdleTime.
	// Therefore: Total Time = Kernel + User.
	dIdle := idle2 - idle1
	dKernel := kernel2 - kernel1
	dUser := user2 - user1
	dTotal := dKernel + dUser

	if dTotal == 0 {
		return 0, false
	}

	// Prevent edge-case rounding glitches under virtualization
	if dIdle > dTotal {
		dIdle = dTotal
	}

	return (1.0 - float64(dIdle)/float64(dTotal)) * 100, true
}

// uptimeSeconds retrieves the number of seconds elapsed since Windows booted.
func uptimeSeconds() int64 {
	// GetTickCount64 returns milliseconds since system startup.
	millis := getTickCountRaw()
	if millis == 0 {
		return 0
	}
	return int64(millis / 1000)
}

// ---- Helper Functions ------------------------------------------------------

// filetimeToUint64 converts Windows dual-32bit DWORDS into a single 64bit jiffy ticker.
func filetimeToUint64(ft syscall.Filetime) uint64 {
	return (uint64(ft.HighDateTime) << 32) | uint64(ft.LowDateTime)
}
