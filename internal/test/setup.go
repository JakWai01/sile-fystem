package internal

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/logging"
	"github.com/JakWai01/sile-fystem/pkg/posix"
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

func (t *TestSetup) Setup(l logging.StructuredLogger, osfs bool) error {
	t.MountConfig.DisableWritebackCaching = true

	cfg := t.MountConfig

	err := t.initialize(context.Background(), &cfg, l, osfs)

	return err
}

func (t *TestSetup) initialize(ctx context.Context, config *fuse.MountConfig, l logging.StructuredLogger, osfs bool) error {
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

	if osfs {
		t.Server = filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), t.Dir, t.TestDir, l, afero.NewOsFs(), false)
	}

	if !osfs {
		t.Server = filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), t.Dir, "/", l, afero.NewMemMapFs(), false)
	}

	t.mfs, err = fuse.Mount(t.Dir, t.Server, config)
	if err != nil {
		return fmt.Errorf("Mount: %v", err)
	}

	return nil
}
