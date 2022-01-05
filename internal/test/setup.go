package internal

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/jacobsa/fuse"
)

type TestSetup struct {
	Server      fuse.Server
	MountConfig fuse.MountConfig
	Ctx         context.Context
	Dir         string
	mfs         *fuse.MountedFileSystem
}

func (t *TestSetup) Setup(server fuse.Server) error {
	t.MountConfig.DisableWritebackCaching = true

	t.Server = server

	cfg := t.MountConfig

	err := t.initialize(context.Background(), t.Server, &cfg)

	return err
}

func (t *TestSetup) initialize(ctx context.Context, server fuse.Server, config *fuse.MountConfig) error {
	t.Ctx = ctx

	if config.OpContext == nil {
		config.OpContext = ctx
	}

	var err error
	t.Dir, err = ioutil.TempDir("", "fuse_test")
	if err != nil {
		return fmt.Errorf("TempDir: %v", err)
	}

	t.mfs, err = fuse.Mount(t.Dir, server, config)
	if err != nil {
		return fmt.Errorf("Mount: %v", err)
	}

	return nil
}
