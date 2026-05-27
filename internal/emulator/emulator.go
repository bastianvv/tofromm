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
	SaveDir(platformFsSlug string) string
	StateDir(platformFsSlug string) string
	ParseSaveName(nameNoExt, ext string) (romFsNameNoExt string, ok bool)
}

type Config struct {
	Platforms []string `mapstructure:"platforms"`
	RomsDir   string   `mapstructure:"roms_dir"`
	SavesDir  string   `mapstructure:"saves_dir"`
	StatesDir string   `mapstructure:"states_dir"`
}

type RomInstaller interface {
	RomExists(platformFsSlug, romFsName string) bool
	InstallRom(platformFsSlug, romFsName, srcPath string) error
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
	case "retroarch-retrodeck":
		return newRetroDeck(cfg), nil
	case "duckstation":
		return newDuckstation(cfg), nil
	case "pcsx2":
		return newPCSX2(cfg), nil
	case "pcsx2-flatpak":
		return newPCSX2Flatpak(cfg), nil
	case "rpcs3":
		return newRPCS3(cfg), nil
	case "rpcs3-flatpak":
		return newRPCS3Flatpak(cfg), nil
	case "dolphin":
		return newDolphin(cfg), nil
	case "duckstation-retrodeck":
		return newDuckstation(cfg), nil
	case "pcsx2-retrodeck":
		return newPCSX2(cfg), nil
	case "rpcs3-retrodeck":
		return newRPCS3(cfg), nil
	case "dolphin-retrodeck":
		return newDolphin(cfg), nil
	default:
		return nil, fmt.Errorf("Unknown emulator: %q - valid options are: retroarch, retroarch-flatpak, retroarch-retrodeck, duckstation, pcsx2, pcsx2-flatpak, rpcs3, rpcs3-flatpak, dolphin, duckstation-retrodeck, pcsx2-retrodeck, rpcs3-retrodeck, dolphin-retrodeck", kind)
	}
}
