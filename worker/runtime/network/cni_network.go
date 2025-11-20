package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	gocni "github.com/containerd/go-cni"
	"github.com/containernetworking/cni/pkg/types"
)

type CNINetworkConfig struct {
	CNIVersion  string
	BridgeName  string
	NetworkName string
	MTU         int
	IPv4        CNIv4NetworkConfig
	IPv6        CNIv6NetworkConfig
	Firewall    CNIFirewallConfig
}

type CNIv4NetworkConfig struct {
	Subnet string
}

type CNIv6NetworkConfig struct {
	Enabled bool
	Subnet  string
	IPMasq  bool
}

type CNIFirewallConfig struct {
	IPTablesAdminChainName string
}

const (
	defaultNetworkName       = "minici"
	defaultPluginDir         = "/opt/cni/bin"
	defaultIPTablesChainName = "MINICI-WORKER"
)

var (
	defaultCNINetworkConfig = CNINetworkConfig{
		NetworkName: defaultNetworkName,
		CNIVersion:  "1.1.0",
		BridgeName:  "minici0",
		IPv4: CNIv4NetworkConfig{
			Subnet: "172.16.0.0/16",
		},
		IPv6: CNIv6NetworkConfig{
			Enabled: false,
			Subnet:  "fd9c:31a6:c759::/64",
			IPMasq:  true,
		},
	}

	defaultFirewallPlugin = FirewallPlugin{
		Plugin:            Plugin{"firewall"},
		IPTablesChainName: defaultIPTablesChainName,
	}

	_, defaultRouteV4, _ = net.ParseCIDR("0.0.0.0/0")

	_, defaultRouteV6, _ = net.ParseCIDR("::/0")
)

type Plugin struct {
	Type string `json:"type"`
}

type CNINetworkConfiguration struct {
	Name       string `json:"name"`
	CNIVersion string `json:"cniVersion"`
	Plugins    []any  `json:"plugins"`
}

type BridgePlugin struct {
	Plugin
	Bridge    string `json:"bridge,omitempty"`
	IsGateway bool   `json:"isGateway,omitempty"`
	IPMasq    bool   `json:"ipMasq,omitempty"`
	IPAM      IPAM   `json:"ipam,omitzero"`
	MTU       int    `json:"mtu,omitempty"`
}

type IPAM struct {
	Type   string        `json:"type,omitempty"`
	Ranges [][]Range     `json:"ranges,omitempty"`
	Routes []types.Route `json:"routes,omitempty"`
}

type FirewallPlugin struct {
	Plugin
	IPTablesChainName string `json:"iptablesAdminChainName"`
}

type Range struct {
	Subnet types.IPNet `json:"subnet"`
}

func (c CNINetworkConfig) ToJSONv4() string {
	_, subnet, err := net.ParseCIDR(c.IPv4.Subnet)
	if err != nil {
		_, subnet, _ = net.ParseCIDR(defaultCNINetworkConfig.IPv4.Subnet)
	}

	ranges := [][]Range{
		{{types.IPNet(*subnet)}},
	}

	routes := []types.Route{
		{Dst: *defaultRouteV4},
		{Dst: *subnet},
	}

	bridgePlugin := BridgePlugin{
		Plugin:    Plugin{"bridge"},
		Bridge:    c.BridgeName,
		IsGateway: true,
		IPMasq:    true,
		MTU:       c.MTU,
		IPAM: IPAM{
			Type:   "host-local",
			Ranges: ranges,
			Routes: routes,
		},
	}

	firewallPlugin := defaultFirewallPlugin
	if c.Firewall.IPTablesAdminChainName != "" {
		firewallPlugin.IPTablesChainName = c.Firewall.IPTablesAdminChainName
	}

	netConfig := CNINetworkConfiguration{
		Name:       c.NetworkName,
		CNIVersion: c.CNIVersion,
		Plugins: []any{
			bridgePlugin,
			firewallPlugin,
		},
	}

	config, _ := json.Marshal(netConfig)

	return string(config)
}

func (c CNINetworkConfig) ToJSONv6() string {
	_, subnet, err := net.ParseCIDR(c.IPv6.Subnet)
	if err != nil {
		_, subnet, _ = net.ParseCIDR(defaultCNINetworkConfig.IPv6.Subnet)
	}

	ranges := [][]Range{
		{{Subnet: types.IPNet(*subnet)}},
	}

	routes := []types.Route{
		{Dst: *defaultRouteV6},
		{Dst: *subnet},
	}

	bridgePlugin := BridgePlugin{
		Plugin:    Plugin{"bridge"},
		Bridge:    c.BridgeName,
		IsGateway: true,
		IPMasq:    c.IPv6.IPMasq,
		MTU:       c.MTU,
		IPAM: IPAM{
			Type:   "host-local",
			Ranges: ranges,
			Routes: routes,
		},
	}

	firewallPlugin := defaultFirewallPlugin
	if c.Firewall.IPTablesAdminChainName != "" {
		firewallPlugin.IPTablesChainName = c.Firewall.IPTablesAdminChainName
	}

	netConfig := CNINetworkConfiguration{
		Name:       c.NetworkName,
		CNIVersion: c.CNIVersion,
		Plugins: []any{
			bridgePlugin,
			firewallPlugin,
		},
	}

	config, _ := json.Marshal(netConfig)

	return string(config)
}

type CNINetworkOpt func(n *cniNetwork)

func WithCNI(cni gocni.CNI) CNINetworkOpt {
	return func(n *cniNetwork) {
		n.cni = cni
	}
}

func WithPluginDir(pluginDir string) CNINetworkOpt {
	return func(n *cniNetwork) {
		n.pluginDir = pluginDir
	}
}

func WithCNINetworkConfig(config CNINetworkConfig) CNINetworkOpt {
	return func(n *cniNetwork) {
		n.config = config
	}
}

type cniNetwork struct {
	cni       gocni.CNI
	config    CNINetworkConfig
	pluginDir string
}

func NewCNINetwork(opts ...CNINetworkOpt) (n *cniNetwork, err error) {
	defer func() {
		if err != nil {
			err = errors.Join(ErrCNIInitFailed, err)
		}
	}()

	n = &cniNetwork{}
	for _, opt := range opts {
		opt(n)
	}

	if n.pluginDir == "" {
		n.pluginDir = defaultPluginDir
	}

	if n.cni == nil {
		n.cni, err = gocni.New(gocni.WithPluginDir([]string{n.pluginDir}))
		if err != nil {
			return nil, err
		}

		opts := []gocni.Opt{
			gocni.WithConfListBytes([]byte(n.config.ToJSONv4())),
			gocni.WithLoNetwork,
		}
		if n.config.IPv6.Enabled {
			opts = append(opts, gocni.WithConfListBytes([]byte(n.config.ToJSONv6())))
		}

		if err = n.cni.Load(opts...); err != nil {
			return nil, fmt.Errorf("load cni network configuration: %w", err)
		}
	}

	return n, nil
}

func (n cniNetwork) Add(ctx context.Context, id string, netNsPath string) (*gocni.Result, error) {
	result, err := n.cni.Setup(ctx, id, netNsPath)
	if err != nil {
		return nil, errors.Join(ErrCNIAddFailed, err)
	}

	return result, nil
}

func (n cniNetwork) Remove(ctx context.Context, id string, netNsPath string) error {
	if err := n.cni.Remove(ctx, id, netNsPath); err != nil {
		return errors.Join(ErrCNIRemoveFailed, err)
	}

	return nil
}

func (n cniNetwork) Check(ctx context.Context, id string, netNsPath string) error {
	if err := n.cni.Check(ctx, id, netNsPath); err != nil {
		return errors.Join(ErrCNICheckFailed, err)
	}

	return nil
}
