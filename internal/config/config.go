package config

import (
	"os"
	"path/filepath"
)

func FilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "tofromm", "config.yaml")
}

func EnsureDir() error {
	return os.MkdirAll(filepath.Dir(FilePath()), 0755)
}
