package fsutil

import (
	"os"
	"path/filepath"
)

type File interface {
	Write(p []byte) (n int, err error)
	Close() error
	Chmod(mode os.FileMode) error
	Name() string
}

type FS interface {
	ReadFile(name string) ([]byte, error)
	Stat(name string) (os.FileInfo, error)
	CreateTemp(dir, pattern string) (File, error)
	Remove(name string) error
	Rename(oldpath, newpath string) error
	Chmod(name string, mode os.FileMode) error
	Glob(pattern string) ([]string, error)
}

type OSFS struct{}

func (OSFS) ReadFile(name string) ([]byte, error)  { return os.ReadFile(name) }
func (OSFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (OSFS) CreateTemp(dir, pattern string) (File, error) {
	return os.CreateTemp(dir, pattern)
}
func (OSFS) Remove(name string) error                  { return os.Remove(name) }
func (OSFS) Rename(oldpath, newpath string) error      { return os.Rename(oldpath, newpath) }
func (OSFS) Chmod(name string, mode os.FileMode) error { return os.Chmod(name, mode) }
func (OSFS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
