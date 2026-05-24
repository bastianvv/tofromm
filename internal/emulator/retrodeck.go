package emulator

import "path/filepath"

type retroDeck struct {
	cfg Config
}

func newRetroDeck(cfg Config) Emulator {
	return &retroDeck{cfg: cfg}
}

func (r *retroDeck) RomPath(platformFsSlug, romFileName string) string {
	return filepath.Join(ExpandPath(r.cfg.RomsDir), platformFsSlug, romFileName)
}

func (r *retroDeck) SavePath(platformFsSlug, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.SavesDir), platformFsSlug, romFsNameNoExt+"."+saveExtension)
}
