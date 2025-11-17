package runtime

import "github.com/containerd/errdefs"

var (
	ErrAborted         = errdefs.ErrAborted
	ErrInternal        = errdefs.ErrInternal
	ErrNotImplemented  = errdefs.ErrNotImplemented
	ErrInvalidArgument = errdefs.ErrInvalidArgument

	ErrNilTask            = ErrInvalidArgument.WithMessage("nil task")
	ErrMissingContainerID = ErrInvalidArgument.WithMessage("missing container id")
	ErrTaskKillTimeout    = ErrAborted.WithMessage("kill timeout exceeded")
	ErrCNIOperationFailed = ErrInternal.WithMessage("cni operation failed")
)
