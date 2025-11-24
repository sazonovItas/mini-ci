package runtime

import (
	"io"
	"sync"

	"github.com/containerd/containerd/v2/pkg/cio"
)

type TaskIO struct {
	StdoutR io.Reader
	StdoutW io.WriteCloser
	StderrR io.Reader
	StderrW io.WriteCloser
}

type IOs map[string]TaskIO

type ContainerIOs map[string]cio.IO

type ioManager struct {
	ios  IOs
	cios ContainerIOs
	lock sync.Mutex
}

var _ IOManager = (*ioManager)(nil)

func NewIOManager() *ioManager {
	return &ioManager{
		ios:  IOs{},
		cios: ContainerIOs{},
		lock: sync.Mutex{},
	}
}

func (iom *ioManager) Creator(id string, creator cio.Creator) cio.Creator {
	return func(id string) (cio.IO, error) {
		iom.lock.Lock()
		defer iom.lock.Unlock()
		prevIO, exists := iom.cios[id]

		newCIO, err := creator(id)
		if newCIO != nil {
			iom.cios[id] = newCIO

			if exists && prevIO != nil {
				prevIO.Cancel()
			}
		}

		return newCIO, err
	}
}

func (iom *ioManager) TaskIO(id string) TaskIO {
	iom.lock.Lock()
	defer iom.lock.Unlock()

	taskIO, ok := iom.ios[id]
	if ok {
		return taskIO
	}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	taskIO = TaskIO{
		StdoutR: stdoutR,
		StdoutW: stdoutW,
		StderrR: stderrR,
		StderrW: stderrW,
	}

	iom.ios[id] = taskIO

	return taskIO
}

func (iom *ioManager) Delete(id string) {
	iom.lock.Lock()
	defer iom.lock.Unlock()

	if taskIO, exists := iom.ios[id]; exists {
		_, _ = taskIO.StdoutW.Close(), taskIO.StderrW.Close()
	}

	delete(iom.ios, id)
	delete(iom.cios, id)
}
