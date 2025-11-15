package filesystem

import "github.com/containerd/errdefs"

var (
	ErrInternal = errdefs.ErrInternal

	ErrLockFail   = ErrInternal.WithMessage("failed to acquire lock")
	ErrUnlockFail = ErrInternal.WithMessage("failed to release lock")
	ErrLockIsNil  = ErrInternal.WithMessage("nil lock")
)
