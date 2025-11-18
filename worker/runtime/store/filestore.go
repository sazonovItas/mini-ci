package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sazonovItas/mini-ci/worker/runtime/filesystem"
)

const (
	defaultFilePerm = 0o600
	defaultDirPerm  = 0o700
)

func NewFileStore(rootPath string, dirPerm os.FileMode, filePerm os.FileMode) (*fileStore, error) {
	if rootPath == "" {
		return nil, ErrRootPathCannotBeEmpty
	}

	if dirPerm == 0 {
		dirPerm = defaultDirPerm
	}

	if filePerm == 0 {
		filePerm = defaultFilePerm
	}

	if err := os.MkdirAll(rootPath, dirPerm); err != nil {
		return nil, errors.Join(ErrSystemFailure, err)
	}

	fileStore := &fileStore{
		dir:      rootPath,
		dirPerm:  dirPerm,
		filePerm: filePerm,
	}

	return fileStore, nil
}

type fileStore struct {
	dir      string
	dirPerm  os.FileMode
	filePerm os.FileMode
}

func (s *fileStore) Exists(keys ...string) (bool, error) {
	path := s.Location(keys...)

	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, errors.Join(ErrSystemFailure, err)
	}

	return true, nil
}

func (s *fileStore) Get(keys ...string) ([]byte, error) {
	path := s.Location(keys...)

	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.Join(ErrNotFound, fmt.Errorf("%q does not exist", path))
		}

		return nil, errors.Join(ErrSystemFailure, err)
	}

	if stat.IsDir() {
		return nil, errors.Join(ErrInternal, fmt.Errorf("%q is a directory and cannot be read as a file", path))
	}

	content, err := filesystem.ReadFile(path)
	if err != nil {
		return nil, errors.Join(ErrSystemFailure, err)
	}

	return content, nil
}

func (s *fileStore) List(keys ...string) ([]string, error) {
	path := s.Location(keys...)

	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.Join(ErrNotFound, err)
		}

		return nil, errors.Join(ErrSystemFailure, err)
	}

	if !stat.IsDir() {
		return nil, errors.Join(ErrInternal, fmt.Errorf("%q is not a directory and cannot be listed", path))
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, errors.Join(ErrSystemFailure, err)
	}

	entries := make([]string, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		entries = append(entries, dirEntry.Name())
	}

	return entries, nil
}

func (s *fileStore) Set(content []byte, keys ...string) error {
	if len(keys) > 1 {
		parent := s.Location(keys[0 : len(keys)-1]...)

		if err := os.MkdirAll(parent, s.dirPerm); err != nil {
			return errors.Join(ErrSystemFailure, err)
		}
	}

	dest := s.Location(keys...)

	stat, err := os.Stat(dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Join(ErrSystemFailure, err)
	}

	if err == nil {
		if stat.IsDir() {
			return errors.Join(ErrInternal, fmt.Errorf("%q is a directory and cannot be written to", dest))
		}
	}

	if err := filesystem.WriteFile(dest, content, s.filePerm); err != nil {
		return errors.Join(ErrSystemFailure, err)
	}

	return nil
}

func (s *fileStore) Append(content []byte, keys ...string) error {
	if len(keys) > 1 {
		parent := s.Location(keys[0 : len(keys)-1]...)

		if err := os.MkdirAll(parent, s.dirPerm); err != nil {
			return errors.Join(ErrSystemFailure, err)
		}
	}

	dest := s.Location(keys...)

	stat, err := os.Stat(dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Join(ErrSystemFailure, err)
	}

	if err != nil && stat.IsDir() {
		return errors.Join(ErrInternal, fmt.Errorf("%q is a directory and cannot be written to", dest))
	}

	file, err := os.OpenFile(dest, os.O_CREATE|os.O_APPEND, s.filePerm)
	if err != nil {
		return errors.Join(ErrSystemFailure, err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Write(content); err != nil {
		return errors.Join(ErrSystemFailure, err)
	}

	return nil
}

func (s *fileStore) Delete(keys ...string) error {
	path := s.Location(keys...)

	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.Join(ErrNotFound, err)
		}

		return errors.Join(ErrSystemFailure, err)
	}

	if stat.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return errors.Join(ErrSystemFailure, err)
		}
	}

	if err := os.Remove(path); err != nil {
		return errors.Join(ErrSystemFailure, err)
	}

	return nil
}

func (s *fileStore) Location(keys ...string) string {
	return filepath.Join(append([]string{s.dir}, keys...)...)
}
