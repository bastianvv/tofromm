package emulator

import "path/filepath"

type retroArch struct {
	cfg Config
}

func newRetroArch(cfg Config) Emulator {
	return &retroArch{cfg: cfg}
}

func (r *retroArch) RomPath(platformFsSlug, romFileName string) string {
	return filepath.Join(ExpandPath(r.cfg.RomsDir), platformFsSlug, romFileName)
}

func (r *retroArch) SavePath(platformFsSlug, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.SavesDir), platformFsSlug, romFsNameNoExt+"."+saveExtension)
}

func (r *retroArch) StatePath(platformFsSlug, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.StatesDir), platformFsSlug, romFsNameNoExt+"."+stateExtension)
}
