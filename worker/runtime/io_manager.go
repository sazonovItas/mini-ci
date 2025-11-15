package runtime

import (
	"sync"

	"github.com/containerd/containerd/v2/pkg/cio"
)

type ContainerIOs map[string]cio.IO

type ioManager struct {
	ioReaders ContainerIOs
	lock      sync.Mutex
}

var _ IOManager = (*ioManager)(nil)

func NewIOManager() *ioManager {
	return &ioManager{
		ioReaders: ContainerIOs{},
		lock:      sync.Mutex{},
	}
}

func (io *ioManager) Creator(containerID string, creator cio.Creator) cio.Creator {
	return func(id string) (cio.IO, error) {
		io.lock.Lock()
		defer io.lock.Unlock()
		prevIO, exists := io.ioReaders[containerID]

		newCIO, err := creator(containerID)
		if newCIO != nil {
			io.ioReaders[containerID] = newCIO

			if exists && prevIO != nil {
				prevIO.Cancel()
			}
		}

		return newCIO, err
	}
}

func (io *ioManager) Attach(containerID string, attach cio.Attach) cio.Attach {
	return func(f *cio.FIFOSet) (cio.IO, error) {
		io.lock.Lock()
		defer io.lock.Unlock()
		prevIO, exists := io.ioReaders[containerID]

		newCIO, err := attach(f)
		if newCIO != nil {
			io.ioReaders[containerID] = newCIO

			if exists && prevIO != nil {
				prevIO.Cancel()
			}
		}

		return newCIO, err
	}
}

func (io *ioManager) Get(containerID string) (cio.IO, bool) {
	io.lock.Lock()
	defer io.lock.Unlock()
	cIO, exists := io.ioReaders[containerID]
	return cIO, exists
}

func (io *ioManager) Delete(containerID string) {
	io.lock.Lock()
	defer io.lock.Unlock()
	delete(io.ioReaders, containerID)
}
