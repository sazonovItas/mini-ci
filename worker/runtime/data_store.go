package runtime

import (
	"github.com/sazonovItas/mini-ci/worker/runtime/idgen"
	"github.com/sazonovItas/mini-ci/worker/runtime/store"
)

const (
	nameFile      = "name"
	containersDir = "containers"
)

type dataStore struct {
	safeStore store.SafeStore
}

var _ DataStore = (*dataStore)(nil)

func NewDataStore(dataStorePath string) (*dataStore, error) {
	safeStore, err := store.NewSafeStore(dataStorePath, 0, 0)
	if err != nil {
		return nil, err
	}

	data := &dataStore{
		safeStore: safeStore,
	}

	return data, nil
}

func (ds *dataStore) NewContainer(id string) error {
	return ds.safeStore.WithLock(func() error {
		err := ds.safeStore.Set([]byte(idgen.ShortID(id)), containersDir, id, nameFile)
		if err != nil {
			return err
		}

		return nil
	})
}

func (ds *dataStore) CleanupContainer(id string) error {
	return ds.safeStore.WithLock(func() error {
		err := ds.safeStore.Delete(containersDir, id)
		if err != nil {
			return err
		}

		return nil
	})
}

func (ds *dataStore) Location(id string) string {
	return ds.safeStore.Location(containersDir, id)
}
