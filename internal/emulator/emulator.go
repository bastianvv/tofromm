package emulator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Emulator interface {
	RomPath(platformFsSlug, romFileName string) string
	SavePath(platformFsSlug, romFsNameNoExt, saveExtension string) string
	StatePath(platformFsSlug, romFsNameNoExt, stateExtension string) string
}

type Config struct {
	RomsDir   string `mapstructure:"roms_dir"`
	SavesDir  string `mapstructure:"saves_dir"`
	StatesDir string `mapstructure:"states_dir"`
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func EnsureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

func New(kind string, cfg Config) (Emulator, error) {
	switch kind {
	case "retroarch":
		return newRetroArch(cfg), nil
	case "retroarch-flatpak":
		return newRetroArchFlatpak(cfg), nil
	case "retrodeck":
		return newRetroDeck(cfg), nil
	default:
		return nil, fmt.Errorf("Unknown emulator: %q - valid options are: retroarch, retroarch-flatpak, retrodeck", kind)
	}
}
