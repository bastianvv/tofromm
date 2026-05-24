package emulator

import (
	"path/filepath"
	"regexp"
)

type duckstation struct {
	cfg Config
}

func newDuckstation(cfg Config) Emulator {
	return &duckstation{
		cfg: cfg,
	}
}

func (d *duckstation) RomPath(platformFsSlug, romFileName string) string {
	return filepath.Join(ExpandPath(d.cfg.RomsDir), romFileName)
}

func (d *duckstation) SavePath(platformFsSlug, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(d.cfg.SavesDir), romFsNameNoExt+"."+saveExtension)
}

func (d *duckstation) StatePath(platformFsSlug, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(d.cfg.StatesDir), romFsNameNoExt+"."+stateExtension)
}

func (d *duckstation) SaveDir(platformFsSlug string) string {
	return ExpandPath(d.cfg.SavesDir)
}

func (d *duckstation) StateDir(platformFsSlug string) string {
	return ExpandPath(d.cfg.StatesDir)
}

func (d *duckstation) ParseSaveName(nameNoExt, ext string) (string, bool) {
	if ext == ".mcd" {
		return mcdSlotSuffix.ReplaceAllString(nameNoExt, ""), true
	}
	return "", false
}

var mcdSlotSuffix = regexp.MustCompile(`_\d+$`)
