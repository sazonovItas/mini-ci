package network

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	containerd "github.com/containerd/containerd/v2/client"
	gocni "github.com/containerd/go-cni"
	"github.com/sazonovItas/mini-ci/worker/runtime/filesystem"
)

type CNINetwork interface {
	Add(ctx context.Context, id string, pid uint32) (result *gocni.Result, err error)
	Check(ctx context.Context, id string, pid uint32) error
	Remove(ctx context.Context, id string, pid uint32) error
}

type ContainerStore interface {
	Location(id string) string
}

type CNIStore interface {
	DeleteResult(id string, inf string) error
	DeleteIPReservation(ip string) error
}

type NetworkOpt func(n *network)

func WithCNINetwork(cni CNINetwork) NetworkOpt {
	return func(n *network) {
		n.cni = cni
	}
}

func WithCNIStore(store CNIStore) NetworkOpt {
	return func(n *network) {
		n.cniStore = store
	}
}

func WithContainerStore(store ContainerStore) NetworkOpt {
	return func(n *network) {
		n.ctrStore = store
	}
}

const (
	networkConfigFile = "network-config.json"
)

type network struct {
	cni      CNINetwork
	cniStore CNIStore
	ctrStore ContainerStore
}

func NewNetwork(opts ...NetworkOpt) (n *network, err error) {
	n = &network{}
	for _, opt := range opts {
		opt(n)
	}

	if n.cni == nil {
		n.cni, err = NewCNINetwork(WithCNINetworkConfig(defaultCNINetworkConfig))
		if err != nil {
			return nil, err
		}
	}

	if n.cniStore == nil {
		n.cniStore, err = NewCNIStore(defaultNetworkName, defaultCNIDir)
		if err != nil {
			return nil, err
		}
	}

	if n.ctrStore == nil {
		return nil, ErrInternal.WithMessage("nil container store")
	}

	return n, nil
}

func (n network) Add(ctx context.Context, id string, task containerd.Task) (err error) {
	result, err := n.cni.Add(ctx, id, task.Pid())
	if err != nil {
		return err
	}

	netConfig, err := json.Marshal(result)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	err = filesystem.WriteFile(n.getNetworkPath(id), netConfig, 0)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	return nil
}

func (n network) Remove(ctx context.Context, id string, task containerd.Task) error {
	if err := n.cni.Remove(ctx, id, task.Pid()); err != nil {
		return err
	}

	return nil
}

func (n network) Cleanup(ctx context.Context, id string) (err error) {
	defer func() {
		if err != nil {
			err = errors.Join(ErrInternal, err)
		}
	}()

	content, err := filesystem.ReadFile(n.getNetworkPath(id))
	if err != nil {
		return err
	}

	var netConfig gocni.Result
	if err := json.Unmarshal(content, &netConfig); err != nil {
		return err
	}

	for inf, config := range netConfig.Interfaces {
		if config == nil {
			continue
		}

		for _, ip := range config.IPConfigs {
			if ip == nil {
				continue
			}

			if err := n.cniStore.DeleteIPReservation(ip.IP.String()); err != nil {
				return err
			}
		}

		if err := n.cniStore.DeleteResult(id, inf); err != nil {
			return ErrInternal
		}
	}

	return nil
}

func (n network) getNetworkPath(id string) string {
	return filepath.Join(n.ctrStore.Location(id), networkConfigFile)
}
