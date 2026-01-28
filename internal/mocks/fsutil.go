package mocks

import (
	"os"

	"integration-suricata-ndpi/pkg/fsutil"
)

type File struct {
	WriteFunc func(p []byte) (n int, err error)
	CloseFunc func() error
	ChmodFunc func(mode os.FileMode) error
	NameFunc  func() string
}

func (f *File) Write(p []byte) (n int, err error) {
	if f.WriteFunc != nil {
		return f.WriteFunc(p)
	}
	return len(p), nil
}

func (f *File) Close() error {
	if f.CloseFunc != nil {
		return f.CloseFunc()
	}
	return nil
}

func (f *File) Chmod(mode os.FileMode) error {
	if f.ChmodFunc != nil {
		return f.ChmodFunc(mode)
	}
	return nil
}

func (f *File) Name() string {
	if f.NameFunc != nil {
		return f.NameFunc()
	}
	return "mock"
}

type FS struct {
	ReadFileFunc   func(name string) ([]byte, error)
	StatFunc       func(name string) (os.FileInfo, error)
	CreateTempFunc func(dir, pattern string) (fsutil.File, error)
	RemoveFunc     func(name string) error
	RenameFunc     func(oldpath, newpath string) error
	ChmodFunc      func(name string, mode os.FileMode) error
	GlobFunc       func(pattern string) ([]string, error)
}

func (m *FS) ReadFile(name string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(name)
	}
	return nil, nil
}

func (m *FS) Stat(name string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m *FS) CreateTemp(dir, pattern string) (fsutil.File, error) {
	if m.CreateTempFunc != nil {
		return m.CreateTempFunc(dir, pattern)
	}
	return &File{}, nil
}

func (m *FS) Remove(name string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(name)
	}
	return nil
}

func (m *FS) Rename(oldpath, newpath string) error {
	if m.RenameFunc != nil {
		return m.RenameFunc(oldpath, newpath)
	}
	return nil
}

func (m *FS) Chmod(name string, mode os.FileMode) error {
	if m.ChmodFunc != nil {
		return m.ChmodFunc(name, mode)
	}
	return nil
}

func (m *FS) Glob(pattern string) ([]string, error) {
	if m.GlobFunc != nil {
		return m.GlobFunc(pattern)
	}
	return nil, nil
}
