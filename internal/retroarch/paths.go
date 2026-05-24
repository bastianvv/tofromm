package retroarch

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	RomsDir  string `mapstructure:"roms_dir"`
	SavesDir string `mapstructure:"saves_dir"`
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func (c *Config) RomPath(platformFsSlug, romFileName string) string {
	return filepath.Join(ExpandPath(c.RomsDir), platformFsSlug, romFileName)
}

func (c *Config) SavePath(platformFsSlug, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(c.SavesDir), platformFsSlug, romFsNameNoExt+"."+saveExtension)
}

func EnsureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}
