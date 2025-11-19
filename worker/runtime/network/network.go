package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/pkg/netns"
	gocni "github.com/containerd/go-cni"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sazonovItas/mini-ci/worker/runtime/idgen"
)

type CNINetwork interface {
	Add(ctx context.Context, id string, netNsPath string) (result *gocni.Result, err error)
	Check(ctx context.Context, id string, netNsPath string) error
	Remove(ctx context.Context, id string, netNsPath string) error
}

type NetworkOpt func(n *network)

func WithCNINetwork(cni CNINetwork) NetworkOpt {
	return func(n *network) {
		n.cni = cni
	}
}

const (
	containersDir = "containers"
	netNsBaseDir  = "/var/run/netns"

	privateFilePerm = os.FileMode(0o600)
)

type network struct {
	cni       CNINetwork
	dataStore string
}

type NetworkConfig struct {
	NetNsPath  string              `json:"netNsPath,omitempty"`
	Interfaces map[string][]net.IP `json:"interfaces,omitempty"`
}

func NewNetwork(dataStore string, opts ...NetworkOpt) (n *network, err error) {
	n = &network{dataStore: dataStore}
	for _, opt := range opts {
		opt(n)
	}

	if n.cni == nil {
		n.cni, err = NewCNINetwork(WithCNINetworkConfig(defaultCNINetworkConfig))
		if err != nil {
			return nil, err
		}
	}

	return n, nil
}

func (n network) Setup(ctx context.Context, id string) (NetworkConfig, error) {
	ns, err := n.setupNetNs()
	if err != nil {
		return NetworkConfig{}, err
	}

	result, err := n.cni.Add(ctx, id, ns.GetPath())
	if err != nil {
		return NetworkConfig{}, err
	}

	if err := n.setupContainerFiles(id); err != nil {
		return NetworkConfig{}, err
	}

	config, err := networkConfig(ns.GetPath(), result)
	if err != nil {
		return NetworkConfig{}, nil
	}

	if err := n.saveConfig(id, config); err != nil {
		return NetworkConfig{}, err
	}

	return config, nil
}

func (n network) setupNetNs() (*netns.NetNS, error) {
	if err := os.MkdirAll(netNsBaseDir, os.FileMode(0o711)); err != nil {
		return nil, errors.Join(ErrMkdirNetNsBaseDir, err)
	}

	ns, err := netns.NewNetNS(netNsBaseDir)
	if err != nil {
		return nil, errors.Join(ErrNewNetNs, err)
	}

	return ns, nil
}

func (n network) setupContainerFiles(id string) error {
	if err := os.MkdirAll(filepath.Join(n.dataStore, containersDir, id), (0o711)); err != nil {
		return err
	}

	if err := n.setupHostname(id); err != nil {
		return err
	}

	if err := n.setupHosts(id); err != nil {
		return err
	}

	if err := n.setupResolvConf(id); err != nil {
		return err
	}

	return nil
}

func (n network) setupHostname(id string) error {
	err := os.WriteFile(n.getHostNamePath(id), []byte(idgen.ShortID(id)), privateFilePerm)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	return nil
}

func (n network) setupHosts(id string) error {
	err := os.WriteFile(n.getHostsPath(id), []byte("127.0.0.1 localhost\n"), privateFilePerm)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	return nil
}

func (n network) setupResolvConf(id string) error {
	content, err := os.ReadFile(resolvConfPath())
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	nameservers := getNameservers(content, IP)

	resolvConf := bytes.NewBuffer(nil)
	for _, ns := range nameservers {
		_, err := resolvConf.Write([]byte("nameserver " + ns + "\n"))
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(n.getResolvConfPath(id), resolvConf.Bytes(), privateFilePerm)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	return nil
}

func (n network) Load(_ context.Context, id string) (NetworkConfig, error) {
	return n.loadConfig(id)
}

func (n network) Cleanup(ctx context.Context, id string) error {
	config, err := n.loadConfig(id)
	if err != nil {
		return err
	}

	if err := n.cni.Remove(ctx, id, config.NetNsPath); err != nil {
		return err
	}

	ns := netns.LoadNetNS(config.NetNsPath)
	if ns != nil {
		if err := ns.Remove(); err != nil {
			return errors.Join(ErrRemoveNetNs, err)
		}
	}

	return nil
}

func (n network) Mounts(id string) []specs.Mount {
	hostname := specs.Mount{
		Type:        "bind",
		Source:      n.getHostNamePath(id),
		Destination: "/etc/hostname",
		Options:     []string{"bind", "rw"},
	}

	hosts := specs.Mount{
		Type:        "bind",
		Source:      n.getHostsPath(id),
		Destination: "/etc/hosts",
		Options:     []string{"bind", "rw"},
	}

	resolvConf := specs.Mount{
		Type:        "bind",
		Source:      n.getResolvConfPath(id),
		Destination: "/etc/resolv.conf",
		Options:     []string{"bind", "rw"},
	}

	return []specs.Mount{hosts, hostname, resolvConf}
}

func networkConfig(netNsPath string, result *gocni.Result) (NetworkConfig, error) {
	if result == nil {
		return NetworkConfig{}, ErrCNINilResult
	}

	interfaces := make(map[string][]net.IP)
	for inf, config := range result.Interfaces {
		if config == nil {
			continue
		}

		var ips []net.IP
		for _, ipConfig := range config.IPConfigs {
			if ipConfig == nil {
				continue
			}

			ips = append(ips, ipConfig.IP)
		}

		interfaces[inf] = ips
	}

	config := NetworkConfig{
		NetNsPath:  netNsPath,
		Interfaces: interfaces,
	}

	return config, nil
}

func (n network) saveConfig(id string, config NetworkConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	err = os.WriteFile(n.getNetworkConfigPath(id), configJSON, privateFilePerm)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	return nil
}

func (n network) loadConfig(id string) (NetworkConfig, error) {
	content, err := os.ReadFile(n.getNetworkConfigPath(id))
	if err != nil {
		return NetworkConfig{}, errors.Join(ErrInternal, err)
	}

	var config NetworkConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return NetworkConfig{}, errors.Join(ErrInternal, err)
	}

	return config, nil
}

func (n network) getNetworkConfigPath(id string) string {
	return filepath.Join(n.dataStore, containersDir, id, "network-config.json")
}

func (n network) getHostNamePath(id string) string {
	return filepath.Join(n.dataStore, containersDir, id, "hostname")
}

func (n network) getHostsPath(id string) string {
	return filepath.Join(n.dataStore, containersDir, id, "hosts")
}

func (n network) getResolvConfPath(id string) string {
	return filepath.Join(n.dataStore, containersDir, id, "resolv.conf")
}
