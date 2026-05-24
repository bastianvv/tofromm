package emulator

import "path/filepath"

type retroArchFlatpak struct {
	cfg Config
}

func newRetroArchFlatpak(cfg Config) Emulator {
	return &retroArchFlatpak{cfg: cfg}
}

func (r *retroArchFlatpak) RomPath(platformFsSlug, romFileName string) string {
	return filepath.Join(ExpandPath(r.cfg.RomsDir), platformFsSlug, romFileName)
}

func (r *retroArchFlatpak) SavePath(platformFsSlug, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.SavesDir), platformFsSlug, romFsNameNoExt+"."+saveExtension)
}
