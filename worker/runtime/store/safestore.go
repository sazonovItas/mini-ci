package store

import (
	"os"
)

func NewSafeStore(rootPath string, dirPerm os.FileMode, filePerm os.FileMode) (*safeStore, error) {
	fileStore, err := NewFileStore(rootPath, dirPerm, filePerm)
	if err != nil {
		return nil, err
	}

	locker := NewLocker(rootPath)

	safeStore := &safeStore{
		locker:    locker,
		fileStore: fileStore,
	}

	return safeStore, nil
}

type safeStore struct {
	*locker
	*fileStore
}

func (s *safeStore) Locker() *locker {
	return s.locker
}

func (s *safeStore) Get(keys ...string) ([]byte, error) {
	if s.locked == nil {
		return nil, ErrMustUseLocker
	}

	return s.Get(keys...)
}

func (s *safeStore) List(keys ...string) ([]string, error) {
	if s.locked == nil {
		return nil, ErrMustUseLocker
	}

	return s.List(keys...)
}

func (s *safeStore) Set(content []byte, keys ...string) error {
	if s.locked == nil {
		return ErrMustUseLocker
	}

	return s.Set(content, keys...)
}

func (s *safeStore) Append(content []byte, keys ...string) error {
	if s.locked == nil {
		return ErrMustUseLocker
	}

	return s.Append(content, keys...)
}

func (s *safeStore) Delete(keys ...string) error {
	if s.locked == nil {
		return ErrMustUseLocker
	}

	return s.Delete(keys...)
}
