package internal

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/helpers"
	"github.com/JakWai01/sile-fystem/pkg/logging"
	"github.com/jacobsa/fuse"
	"github.com/spf13/afero"
)

type TestSetup struct {
	Server      fuse.Server
	MountConfig fuse.MountConfig
	Ctx         context.Context
	Dir         string
	TestDir     string
	mfs         *fuse.MountedFileSystem
}

func (t *TestSetup) Setup(l logging.StructuredLogger, backend afero.Fs) error {
	t.MountConfig.DisableWritebackCaching = true

	cfg := t.MountConfig

	err := t.initialize(context.Background(), &cfg, l, backend)

	return err
}

func (t *TestSetup) initialize(ctx context.Context, config *fuse.MountConfig, l logging.StructuredLogger, backend afero.Fs) error {
	t.Ctx = ctx

	if config.OpContext == nil {
		config.OpContext = ctx
	}

	var err error
	t.Dir, err = ioutil.TempDir("", "fuse_test")
	if err != nil {
		return fmt.Errorf("TempDir: %v", err)
	}

	t.TestDir, err = ioutil.TempDir("", "fuse_test_dir")
	if err != nil {
		return fmt.Errorf("TempDir2: %v", err)
	}

	// OsFs
	t.Server = filesystem.NewFileSystem(helpers.CurrentUid(), helpers.CurrentGid(), t.Dir, t.TestDir, l, backend)

	// MemMapFs
	// t.Server = filesystem.NewFileSystem(helpers.CurrentUid(), helpers.CurrentGid(), t.Dir, "/", l, backend)

	t.mfs, err = fuse.Mount(t.Dir, t.Server, config)
	if err != nil {
		return fmt.Errorf("Mount: %v", err)
	}

	return nil
}
