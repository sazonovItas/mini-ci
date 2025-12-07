package runtime

import (
	"fmt"
	"os"

	"github.com/sazonovItas/mini-ci/worker/runtime/filesystem"
	"github.com/sazonovItas/mini-ci/worker/runtime/store"
)

const (
	containersDir = "containers"
)

type dataStore struct {
	store store.Store
}

var _ DataStore = (*dataStore)(nil)

func NewDataStore(dataStorePath string) (*dataStore, error) {
	fileStore, err := store.NewFileStore(dataStorePath, 0, 0)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(fileStore.Location(containersDir), os.FileMode(0o700)); err != nil {
		return nil, fmt.Errorf("failed create containers directory: %w", err)
	}

	data := &dataStore{
		store: fileStore,
	}

	return data, nil
}

func (ds dataStore) New(id string) error {
	return filesystem.WithLock(ds.store.Location(containersDir), func() error {
		dir := ds.Location(id)

		if err := os.MkdirAll(dir, os.FileMode(0o700)); err != nil {
			return fmt.Errorf("%w: container id - %s: %w", ErrMakeContainerDir, id, err)
		}

		return nil
	})
}

func (ds dataStore) Cleanup(id string) error {
	return filesystem.WithLock(ds.store.Location(containersDir), func() error {
		if err := ds.store.Delete(containersDir, id); err != nil {
			return fmt.Errorf("clean up container dir %s: %w", id, err)
		}

		return nil
	})
}

func (ds dataStore) Location(id string, keys ...string) string {
	return ds.store.Location(ds.pathKeys(id, keys...)...)
}

func (ds dataStore) Get(id string, keys ...string) ([]byte, error) {
	return ds.store.Get(ds.pathKeys(id, keys...)...)
}

func (ds dataStore) Set(content []byte, id string, keys ...string) error {
	return ds.store.Set(content, ds.pathKeys(id, keys...)...)
}

func (ds dataStore) pathKeys(id string, keys ...string) []string {
	return append([]string{containersDir, id}, keys...)
}
