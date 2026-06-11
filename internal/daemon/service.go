package daemon

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const serviceFileName = "io.github.bastianvv.tofromm-daemon.service"

func ServiceInstalled() bool {
	_, err := os.Stat(serviceDestPath())
	return err == nil
}

func InstallService(content []byte) error {
	dest := serviceDestPath()

	if !IsInsideFlatpak() {
		return fmt.Errorf("background sync service is only available via Flatpak")
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create systemd dir: %w", err)
	}

	if err := os.WriteFile(dest, content, 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	return runSystemctl("enable", "--now", serviceFileName)
}

func serviceDestPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", serviceFileName)
}

func runSystemctl(args ...string) error {
	var cmd *exec.Cmd
	if IsInsideFlatpak() {
		cmd = exec.Command("flatpak-spawn", append([]string{"--host", "systemctl", "--user"}, args...)...)
	} else {
		cmd = exec.Command("systemctl", append([]string{"--user"}, args...)...)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}

func IsInsideFlatpak() bool {
	_, err := os.Stat("/.flatpak-info")
	return err == nil
}
