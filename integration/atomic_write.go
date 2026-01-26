package integration

import (
	"os"
	"path/filepath"

	"integration-suricata-ndpi/pkg/fsutil"
)

func writeFileAtomic(path string, data []byte, perm os.FileMode, fs fsutil.FS) error {
	if fs == nil {
		fs = fsutil.OSFS{}
	}
	dir := filepath.Dir(path)

	tmp, err := fs.CreateTemp(dir, ".suricata.yaml.*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = fs.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}

	if f, ok := any(tmp).(interface{ Sync() error }); ok {
		if err := f.Sync(); err != nil {
			_ = tmp.Close()
			return err
		}
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := fs.Rename(tmpName, path); err != nil {
		return err
	}

	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	return nil
}
