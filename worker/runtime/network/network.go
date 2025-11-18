package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	gocni "github.com/containerd/go-cni"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sazonovItas/mini-ci/worker/runtime/filesystem"
	"github.com/sazonovItas/mini-ci/worker/runtime/idgen"
)

type CNINetwork interface {
	Add(ctx context.Context, id string, pid uint32) (result *gocni.Result, err error)
	Check(ctx context.Context, id string, pid uint32) error
	Remove(ctx context.Context, id string, pid uint32) error
}

type ContainerStore interface {
	Get(id string, keys ...string) ([]byte, error)
	Set(content []byte, id string, keys ...string) error
	Location(id string, keys ...string) string
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
	hostsFile         = "hosts"
	hostnameFile      = "hostname"
	resolvConfFile    = "resolv.conf"
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

	if n.ctrStore == nil {
		return nil, ErrInternal.WithMessage("nil container store")
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

	return n, nil
}

func (n network) Init(id string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("init network: %w", err)
		}
	}()

	err = n.ctrStore.Set([]byte{}, id, hostnameFile)
	if err != nil {
		return err
	}

	err = n.ctrStore.Set([]byte{}, id, hostsFile)
	if err != nil {
		return err
	}

	err = n.ctrStore.Set([]byte{}, id, resolvConfFile)
	if err != nil {
		return err
	}

	return nil
}

func (n network) Mounts(id string) []specs.Mount {
	hostname := specs.Mount{
		Type:        "bind",
		Source:      n.ctrStore.Location(id, hostnameFile),
		Destination: "/etc/hostname",
		Options:     []string{"bind", "rw"},
	}

	hosts := specs.Mount{
		Type:        "bind",
		Source:      n.ctrStore.Location(id, hostsFile),
		Destination: "/etc/hosts",
		Options:     []string{"bind", "rw"},
	}

	resolvConf := specs.Mount{
		Type:        "bind",
		Source:      n.ctrStore.Location(id, resolvConfFile),
		Destination: "/etc/resolv.conf",
		Options:     []string{"bind", "rw"},
	}

	return []specs.Mount{hosts, hostname, resolvConf}
}

func (n network) Add(ctx context.Context, id string, task containerd.Task) error {
	if err := n.addHostname(id); err != nil {
		return err
	}

	if err := n.addResolveConf(id); err != nil {
		return err
	}

	if err := n.addCNI(ctx, id, task); err != nil {
		return err
	}

	if err := n.addEtcHosts(id, nil); err != nil {
		return err
	}

	return nil
}

func (n network) addCNI(ctx context.Context, id string, task containerd.Task) error {
	result, err := n.cni.Add(ctx, id, task.Pid())
	if err != nil {
		return err
	}

	interfaces, err := json.Marshal(result.Interfaces)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	err = n.ctrStore.Set(interfaces, id, networkConfigFile)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	return nil
}

func (n network) addHostname(id string) error {
	hostname := idgen.ShortID(id)

	err := n.ctrStore.Set([]byte(hostname), id, hostnameFile)
	if err != nil {
		return fmt.Errorf("add hostname: %w", err)
	}

	return nil
}

func (n network) addResolveConf(id string) error {
	content, err := filesystem.ReadFile(resolvConfPath())
	if err != nil {
		return err
	}

	dns := getNameservers(content, IP)

	resolvConf, err := buildResolvConf(dns)
	if err != nil {
		return err
	}

	err = n.ctrStore.Set([]byte(resolvConf), id, resolvConfFile)
	if err != nil {
		return fmt.Errorf("add resolve config: %w", err)
	}

	return nil
}

func (n network) addEtcHosts(id string, _ map[string]*gocni.Config) error {
	err := n.ctrStore.Set([]byte("127.0.0.1 localhost\n"), id, hostsFile)
	if err != nil {
		return err
	}

	return nil
}

func (n network) Remove(ctx context.Context, id string, task containerd.Task) error {
	if err := n.cni.Remove(ctx, id, task.Pid()); err != nil {
		return err
	}

	return nil
}

func (n network) Cleanup(id string) (err error) {
	if err := n.cleanupCNI(id); err != nil {
		return err
	}

	return nil
}

func (n network) cleanupCNI(id string) (err error) {
	content, err := n.ctrStore.Get(id, networkConfigFile)
	if err != nil {
		return err
	}

	var interfaces map[string]*gocni.Config
	if err := json.Unmarshal(content, &interfaces); err != nil {
		return err
	}

	for inf, config := range interfaces {
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
