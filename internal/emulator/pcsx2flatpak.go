package emulator

import "path/filepath"

type pcsx2Flatpak struct {
	cfg Config
}

func newPCSX2Flatpak(cfg Config) Emulator {
	return &pcsx2Flatpak{cfg: cfg}
}

func (p *pcsx2Flatpak) RomPath(platformFsSlug, romFileName string) string {
	return filepath.Join(ExpandPath(p.cfg.RomsDir), romFileName)
}

func (p *pcsx2Flatpak) SavePath(platformFsSlug, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(p.cfg.SavesDir), romFsNameNoExt+"."+saveExtension)
}

func (p *pcsx2Flatpak) StatePath(platformFsSlug, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(p.cfg.StatesDir), romFsNameNoExt+"."+stateExtension)
}

func (p *pcsx2Flatpak) SaveDir(_ string) string {
	return ExpandPath(p.cfg.SavesDir)
}

func (p *pcsx2Flatpak) StateDir(_ string) string {
	return ExpandPath(p.cfg.StatesDir)
}

func (p *pcsx2Flatpak) ParseSaveName(nameNoExt, ext string) (string, bool) {
	if ext == ".ps2" {
		return nameNoExt, true
	}
	return "", false
}
