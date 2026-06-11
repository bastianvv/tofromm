package daemon

import (
	"io/fs"
	"path/filepath"
	"time"
)

func SnapshotDirs(dirs []string) map[string]time.Time {
	snapshot := make(map[string]time.Time)
	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			snapshot[path] = info.ModTime()
			return nil
		})
	}
	return snapshot
}

func SavesChanged(dirs []string, snapshot map[string]time.Time) bool {
	for _, dir := range dirs {
		changed := false
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if changed || err != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			prev, ok := snapshot[path]
			if !ok || info.ModTime().After(prev) {
				changed = true
			}
			return nil
		})
		if changed {
			return true
		}
	}
	return false
}
