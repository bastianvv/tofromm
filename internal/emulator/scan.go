package emulator

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalSave struct {
	RomID         int
	FileName      string
	Dir           string
	ModTime       time.Time
	FileSizeBytes int64
	IsState       bool
}

// ScanSaves scans save and state directories for all given platforms.
// romIndex maps fs_name_no_ext to rom ID.
func ScanSaves(emu Emulator, platforms []string, romIndex map[string]int) []LocalSave {
	var results []LocalSave

	for _, slug := range platforms {
		saveDir := emu.SaveDir(slug)
		if entries, err := os.ReadDir(saveDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				ext := filepath.Ext(name)
				if ext == "" {
					continue
				}
				nameNoExt := strings.TrimSuffix(name, ext)
				romName, ok := emu.ParseSaveName(nameNoExt, ext)
				if !ok {
					continue
				}
				romID, ok := romIndex[romName]
				if !ok {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					continue
				}
				results = append(results, LocalSave{
					RomID:         romID,
					FileName:      name,
					Dir:           saveDir,
					ModTime:       info.ModTime(),
					FileSizeBytes: info.Size(),
					IsState:       false,
				})
			}
		}

		stateDir := emu.StateDir(slug)
		if entries, err := os.ReadDir(stateDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				ext := filepath.Ext(name)
				if !strings.HasPrefix(ext, ".state") {
					continue
				}
				nameNoExt := strings.TrimSuffix(name, ext)
				romID, ok := romIndex[nameNoExt]
				if !ok {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					continue
				}
				results = append(results, LocalSave{
					RomID:         romID,
					Dir:           stateDir,
					FileName:      name,
					ModTime:       info.ModTime(),
					FileSizeBytes: info.Size(),
					IsState:       true,
				})
			}
		}
	}

	return results
}
