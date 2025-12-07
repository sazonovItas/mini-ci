package network

import "github.com/containerd/errdefs"

var (
	ErrInternal = errdefs.ErrInternal

	ErrCNIInitFailed   = ErrInternal.WithMessage("cni init failed")
	ErrCNIAddFailed    = ErrInternal.WithMessage("cni add failed")
	ErrCNICheckFailed  = ErrInternal.WithMessage("cni check failed")
	ErrCNIRemoveFailed = ErrInternal.WithMessage("cni remove failed")
	ErrCNINilResult    = ErrInternal.WithMessage("cni nil result")

	ErrMkdirNetNsBaseDir = ErrInternal.WithMessage("make net ns base dir failed")
	ErrNewNetNs          = ErrInternal.WithMessage("new net ns failed")
	ErrRemoveNetNs       = ErrInternal.WithMessage("remove net ns failed")

	ErrSetupHosts      = ErrInternal.WithMessage("setup hosts")
	ErrSetupHostname   = ErrInternal.WithMessage("setup hostname")
	ErrSetupResolvConf = ErrInternal.WithMessage("setup resolv conf")

	ErrLoadNetConfig = ErrInternal.WithMessage("load network config")
	ErrSaveNetConfig = ErrInternal.WithMessage("save network config")
)
