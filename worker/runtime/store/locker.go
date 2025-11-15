package store

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sazonovItas/mini-ci/worker/runtime/filesystem"
)

func NewLocker(path string) *locker {
	return &locker{
		path: path,
	}
}

type locker struct {
	mu     sync.Mutex
	locked *os.File
	path   string
}

func (l *locker) Lock() error {
	l.mu.Lock()

	file, err := filesystem.Lock(l.path)
	if err != nil {
		return errors.Join(ErrLockFailure, err)
	}

	l.locked = file

	return nil
}

func (l *locker) Unlock() error {
	if l.locked == nil {
		return errors.Join(ErrCannotUnlockNotLocked, fmt.Errorf("lock is %q", l.path))
	}

	defer l.mu.Unlock()

	defer func() {
		l.locked = nil
	}()

	if err := filesystem.Unlock(l.locked); err != nil {
		return errors.Join(ErrLockFailure, err)
	}

	return nil
}

func (l *locker) WithLock(f func() error) (err error) {
	if err = l.Lock(); err != nil {
		return err
	}

	defer func() {
		err = errors.Join(l.Unlock(), err)
	}()

	return f()
}
