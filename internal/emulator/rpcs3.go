package emulator

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type rpcs3 struct {
	cfg Config
}

func newRPCS3(cfg Config) *rpcs3 {
	return &rpcs3{cfg: cfg}
}

func (r *rpcs3) RomPath(_, romFsName string) string {
	return filepath.Join(ExpandPath(r.cfg.RomsDir), r.gameDirName(romFsName))
}

func (r *rpcs3) SavePath(_, romFsNameNoExt, saveExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.SavesDir), romFsNameNoExt+"."+saveExtension)
}

func (r *rpcs3) StatePath(_, romFsNameNoExt, stateExtension string) string {
	return filepath.Join(ExpandPath(r.cfg.StatesDir), romFsNameNoExt+"."+stateExtension)
}

func (r *rpcs3) SaveDir(_ string) string {
	return ExpandPath(r.cfg.SavesDir)
}

func (r *rpcs3) StateDir(_ string) string {
	return ExpandPath(r.cfg.StatesDir)
}

func (r *rpcs3) ParseSaveName(_, _ string) (string, bool) {
	return "", false
}

func (r *rpcs3) RomExists(_, romFsName string) bool {
	_, err := os.Stat(filepath.Join(ExpandPath(r.cfg.RomsDir), r.gameDirName(romFsName)))
	return err == nil
}

func (r *rpcs3) InstallRom(_, romFsName, srcPath string) error {
	dest := filepath.Join(ExpandPath(r.cfg.RomsDir), r.gameDirName(romFsName))
	return extractZip(srcPath, dest)
}

func (r *rpcs3) gameDirName(romFsName string) string {
	if strings.HasSuffix(romFsName, ".ps3.zip") {
		return strings.TrimSuffix(romFsName, ".ps3.zip")
	}
	return strings.TrimSuffix(romFsName, filepath.Ext(romFsName))
}

func extractZip(src, dest string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	destClean := filepath.Clean(dest)
	for _, f := range zr.File {
		target := filepath.Join(destClean, f.Name)
		if target != destClean && !strings.HasPrefix(target, destClean+string(os.PathSeparator)) {
			return fmt.Errorf("Invalid file path: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
