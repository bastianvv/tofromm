package emulator

import (
	"os"
	"path/filepath"
	"strings"
)

type rpcs3Flatpak struct {
	cfg Config
}

func newRPCS3Flatpak(cfg Config) Emulator {
	return &rpcs3Flatpak{cfg: cfg}
}

func (r *rpcs3Flatpak) RomPath(_, romFsName string) string {
	return filepath.Join(ExpandPath(r.cfg.RomsDir), r.gameDirName(romFsName))
}

func (r *rpcs3Flatpak) SavePath(_, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.SavesDir), romFsNameNoExt+"."+saveExtension)
}

func (r *rpcs3Flatpak) StatePath(_, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.StatesDir), romFsNameNoExt+"."+stateExtension)
}

func (r *rpcs3Flatpak) SaveDir(_ string) string  { return ExpandPath(r.cfg.SavesDir) }
func (r *rpcs3Flatpak) StateDir(_ string) string { return ExpandPath(r.cfg.StatesDir) }

func (r *rpcs3Flatpak) ParseSaveName(_, _ string) (string, bool) { return "", false }

func (r *rpcs3Flatpak) RomExists(_, romFsName string) bool {
	_, err := os.Stat(filepath.Join(ExpandPath(r.cfg.RomsDir), r.gameDirName(romFsName)))
	return err == nil
}

func (r *rpcs3Flatpak) InstallRom(_, romFsName, srcPath string) error {
	dest := filepath.Join(ExpandPath(r.cfg.RomsDir), r.gameDirName(romFsName))
	return extractZip(srcPath, dest)
}

func (r *rpcs3Flatpak) gameDirName(romFsName string) string {
	if strings.HasSuffix(romFsName, ".ps3.zip") {
		return strings.TrimSuffix(romFsName, ".ps3.zip")
	}
	return strings.TrimSuffix(romFsName, filepath.Ext(romFsName))
}
