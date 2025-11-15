package runtime

import (
	"context"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/snapshots"
	"github.com/containerd/errdefs"
	"github.com/opencontainers/image-spec/identity"
)

type rootfsManager struct {
	client *containerd.Client

	imageStore  images.Store
	snapshotter snapshots.Snapshotter
}

var _ RootFSManager = (*rootfsManager)(nil)

func NewRootFSManager(
	client *containerd.Client,
	imageStore images.Store,
	snapshotter snapshots.Snapshotter,
) *rootfsManager {
	return &rootfsManager{
		client:      client,
		imageStore:  imageStore,
		snapshotter: snapshotter,
	}
}

func (rm *rootfsManager) PullImage(ctx context.Context, imageRef string) (containerd.Image, error) {
	return rm.client.Pull(ctx, imageRef, containerd.WithPullUnpack)
}

func (rm *rootfsManager) Image(ctx context.Context, imageRef string) (containerd.Image, error) {
	image, err := rm.imageStore.Get(ctx, imageRef)
	if err != nil {
		return nil, err
	}

	return containerd.NewImage(rm.client, image), nil
}

func (rm *rootfsManager) GetOrPullImage(ctx context.Context, imageRef string) (image containerd.Image, err error) {
	image, err = rm.Image(ctx, imageRef)
	if err == nil {
		return image, nil
	}

	if !errdefs.IsNotFound(err) {
		return nil, err
	}

	image, err = rm.PullImage(ctx, imageRef)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func (rm *rootfsManager) DeleteImage(ctx context.Context, imageRef string) error {
	return rm.imageStore.Delete(ctx, imageRef)
}

func (rm *rootfsManager) Snapshot(ctx context.Context, snapshotKey string) (snapshots.Info, error) {
	return rm.snapshotter.Stat(ctx, snapshotKey)
}

func (rm *rootfsManager) CreateSnapshot(ctx context.Context, image containerd.Image, snapshotKey string) error {
	diffIDs, err := image.RootFS(ctx)
	if err != nil {
		return err
	}

	parent := identity.ChainID(diffIDs).String()
	_, err = rm.snapshotter.Prepare(ctx, snapshotKey, parent)
	if err != nil {
		return err
	}

	return nil
}

func (rm *rootfsManager) DeleteSnapshot(ctx context.Context, snapshotKey string) error {
	return rm.snapshotter.Remove(ctx, snapshotKey)
}
