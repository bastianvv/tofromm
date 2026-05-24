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

func (r *retroArchFlatpak) StatePath(platformFsSlug, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.StatesDir), platformFsSlug, romFsNameNoExt+"."+stateExtension)
}

func (r *retroArchFlatpak) SaveDir(platformFsSlug string) string {
	return filepath.Join(ExpandPath(r.cfg.SavesDir), platformFsSlug)
}

func (r *retroArchFlatpak) StateDir(platformFsSlug string) string {
	return filepath.Join(ExpandPath(r.cfg.StatesDir), platformFsSlug)
}

func (r *retroArchFlatpak) ParseSaveName(nameNoExt, ext string) (string, bool) {
	return nameNoExt, true
}
