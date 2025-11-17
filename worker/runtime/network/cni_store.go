package network

import (
	"strings"

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
	network string
	store   store.Store
}

func NewCNIStore(network string, path string) (*cniStore, error) {
	st, err := store.NewFileStore(path, 0, 0)
	if err != nil {
		return nil, err
	}

	cni := &cniStore{
		store:   st,
		network: network,
	}

	return cni, nil
}

func (cs cniStore) DeleteResult(id, inf string) error {
	err := cs.store.Delete(
		cniResultsDir,
		strings.Join([]string{cs.network, id, inf}, "-"),
	)
	if err != nil {
		return err
	}

	return nil
}

func (cs cniStore) DeleteIPReservation(ip string) error {
	return filesystem.WithLock(cs.getLockPath(), func() error {
		if err := cs.store.Delete(cniNetworksDir, cs.network, ip); err != nil {
			return err
		}

		return nil
	})
}

func (cs cniStore) getLockPath() string {
	return cs.store.Location(cniNetworksDir, cniLockFile)
}
