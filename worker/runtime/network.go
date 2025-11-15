package runtime

import (
	"context"

	containerd "github.com/containerd/containerd/v2/client"
	gocni "github.com/containerd/go-cni"
)

type CNINetwork interface {
	Add(ctx context.Context, id string, task containerd.Task) (result *gocni.Result, err error)
	Check(ctx context.Context, id string, task containerd.Task) error
	Remove(ctx context.Context, id string, task containerd.Task) error
}

type network struct{}

func NewNetwork() *network {
	return &network{}
}

func (n *network) Add(ctx context.Context, id string, task containerd.Task) error {
	return ErrNotImplemented
}

func (n *network) Remove(ctx context.Context, id string, task containerd.Task) error {
	return ErrNotImplemented
}

func (n *network) Cleanup(ctx context.Context, id string) error {
	return ErrNotImplemented
}
