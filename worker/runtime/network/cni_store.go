package network

import (
	"strings"

	"github.com/containerd/errdefs"
	"github.com/sazonovItas/mini-ci/worker/runtime/filesystem"
	"github.com/sazonovItas/mini-ci/worker/runtime/store"
)

const (
	defaultCNIDir = "/var/lib/cni"

	cniResultsDir  = "results"
	cniNetworksDir = "networks"
	cniLockFile    = "lock"
)

type cniStore struct {
	store       store.Store
	networkName string
}

func NewCNIStore(networkName string, cniDir string) (*cniStore, error) {
	st, err := store.NewFileStore(cniDir, 0, 0)
	if err != nil {
		return nil, err
	}

	cni := &cniStore{
		store:       st,
		networkName: networkName,
	}

	return cni, nil
}

func (cs cniStore) DeleteResult(id, inf string) error {
	err := cs.store.Delete(
		cniResultsDir,
		strings.Join([]string{cs.networkName, id, inf}, "-"),
	)
	if err != nil && !errdefs.IsNotFound(err) {
		return err
	}

	return nil
}

func (cs cniStore) DeleteIPReservation(ip string) error {
	return filesystem.WithLock(cs.getLockPath(), func() error {
		err := cs.store.Delete(cniNetworksDir, cs.networkName, ip)
		if err != nil && !errdefs.IsNotFound(err) {
			return err
		}

		return nil
	})
}

func (cs cniStore) getLockPath() string {
	return cs.store.Location(cniNetworksDir, cniLockFile)
}
