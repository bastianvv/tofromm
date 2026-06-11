package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var emulatorStems = []string{
	"retroarch",
	"duckstation",
	"pcsx2",
	"rpcs3",
	"dolphin-emu",
}

func EmulatorRunning() bool {
	if IsInsideFlatpak() {
		return emulatorRunningHost()
	}
	return emulatorRunningProc()
}

func emulatorRunningProc() bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, ok := strconvAllDigits(e.Name()); !ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		if matchesEmulator(string(data)) {
			return true
		}
	}
	return false
}

func emulatorRunningHost() bool {
	out, err := exec.Command("flatpak-spawn", "--host", "sh", "-c", "cat /proc/[0-9]*/comm 2>/dev/null").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if matchesEmulator(line) {
			return true
		}
	}
	return false
}

func matchesEmulator(comm string) bool {
	comm = strings.TrimSpace(comm)
	for _, stem := range emulatorStems {
		if strings.Contains(comm, stem) {
			return true
		}
	}
	return false
}

func strconvAllDigits(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	return 0, true
}
