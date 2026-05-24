package emulator

import "path/filepath"

type dolphin struct {
	cfg Config
}

func newDolphin(cfg Config) Emulator {
	return &dolphin{cfg: cfg}
}

func (d *dolphin) RomPath(_, romFsName string) string {
	return filepath.Join(ExpandPath(d.cfg.RomsDir), romFsName)
}

func (d *dolphin) SavePath(_, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(d.cfg.SavesDir), romFsNameNoExt+"."+saveExtension)
}

func (d *dolphin) StatePath(_, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(d.cfg.StatesDir), romFsNameNoExt+"."+stateExtension)
}

func (d *dolphin) SaveDir(_ string) string  { return ExpandPath(d.cfg.SavesDir) }
func (d *dolphin) StateDir(_ string) string { return ExpandPath(d.cfg.StatesDir) }

func (d *dolphin) ParseSaveName(_, _ string) (string, bool) { return "", false }
