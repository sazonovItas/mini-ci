package store

import "github.com/containerd/errdefs"

var (
	ErrNotFound        = errdefs.ErrNotFound
	ErrInternal        = errdefs.ErrInternal
	ErrInvalidArgument = errdefs.ErrInvalidArgument

	ErrRootPathCannotBeEmpty = ErrInvalidArgument.WithMessage("filestore root path cannot be empty")

	ErrLockFailure           = ErrInternal.WithMessage("lock failure")
	ErrSystemFailure         = ErrInternal.WithMessage("filesystem failure")
	ErrCannotUnlockNotLocked = ErrInternal.WithMessage("store cannot unlock not locked")
	ErrMustUseLocker         = ErrInternal.WithMessage("must use locker in the safe store")
)
