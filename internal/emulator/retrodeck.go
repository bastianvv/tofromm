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

func (r *retroDeck) StatePath(platformFsSlug, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.StatesDir), platformFsSlug, romFsNameNoExt+"."+stateExtension)
}

func (r *retroDeck) SaveDir(platformFsSlug string) string {
	return filepath.Join(ExpandPath(r.cfg.SavesDir), platformFsSlug)
}

func (r *retroDeck) StateDir(platformFsSlug string) string {
	return filepath.Join(ExpandPath(r.cfg.StatesDir), platformFsSlug)
}

func (r *retroDeck) ParseSaveName(nameNoExt, ext string) (string, bool) {
	return nameNoExt, true
}
