package filesystem

import (
	"errors"
	"os"
	"syscall"
)

const (
	privateFilePermission = os.FileMode(0o600)
)

func Lock(path string) (file *os.File, err error) {
	return commonlock(path, writeLock)
}

func RLock(path string) (file *os.File, err error) {
	return commonlock(path, readLock)
}

func Unlock(lock *os.File) error {
	if lock == nil {
		return ErrLockIsNil
	}

	if err := errors.Join(unlock(lock), lock.Close()); err != nil {
		return errors.Join(ErrUnlockFail, err)
	}

	return nil
}

func WithLock(path string, f func() error) (err error) {
	file, err := Lock(path)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(Unlock(file), err)
	}()

	return f()
}

func WithRLock(path string, f func() error) (err error) {
	file, err := RLock(path)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(Unlock(file), err)
	}()

	return f()
}

func commonlock(path string, mode lockMode) (file *os.File, err error) {
	defer func() {
		if err != nil {
			err = errors.Join(ErrLockFail, err)
		}
	}()

	file, err = os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		file, err = os.OpenFile(path, os.O_RDONLY|os.O_CREATE, privateFilePermission)
	}
	if err != nil {
		return nil, err
	}

	if err = lock(file, mode); err != nil {
		return nil, errors.Join(err, file.Close())
	}

	return file, nil
}

type lockMode int

const (
	readLock  lockMode = syscall.LOCK_SH
	writeLock lockMode = syscall.LOCK_EX
)

func lock(file *os.File, mode lockMode) error {
	var err error

	for {
		err = syscall.Flock(int(file.Fd()), int(mode))
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}

	return err
}

func unlock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
