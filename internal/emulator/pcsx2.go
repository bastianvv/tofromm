package emulator

import "path/filepath"

type pcsx2 struct {
	cfg Config
}

func newPCSX2(cfg Config) Emulator {
	return &pcsx2{cfg: cfg}
}

func (p *pcsx2) RomPath(platformSlug, romFileName string) string {
	return filepath.Join(ExpandPath(p.cfg.RomsDir), romFileName)
}

func (p *pcsx2) SavePath(platformSlug, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(p.cfg.SavesDir), romFsNameNoExt+"."+saveExtension)
}

func (p *pcsx2) StatePath(platformSlug, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(p.cfg.StatesDir), romFsNameNoExt+"."+stateExtension)
}

func (p *pcsx2) SaveDir(_ string) string {
	return ExpandPath(p.cfg.SavesDir)
}

func (p *pcsx2) StateDir(_ string) string {
	return ExpandPath(p.cfg.StatesDir)
}

func (p *pcsx2) ParseSaveName(nameNoExt, ext string) (string, bool) {
	if ext == ".ps2" {
		return nameNoExt, true
	}
	return "", false
}
