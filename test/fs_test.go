package filesystem

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JakWai01/sile-fystem/pkg/helpers"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/samples/memfs"
)

const (
	fileMode = 0754
	timeSlop = 25 * time.Millisecond
)

var (
	test TestSetup
)

type TestSetup struct {
	Server      fuse.Server
	MountConfig fuse.MountConfig
	Ctx         context.Context
	Dir         string
	ToClose     []io.Closer
	mfs         *fuse.MountedFileSystem
}

func TestFileSystemSetup(t *testing.T) {
	test = TestSetup{}

	err := test.Setup()
	if err != nil {
		t.Fail()
	}
}

func Test_Mkdir(t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	dirName := path.Join(test.Dir, "dir")

	err = os.Mkdir(dirName, 0754)
	if err != nil {
		t.Fail()
	}

	// Stat the directory.
	fi, err = os.Stat(dirName)
	if err != nil {
		t.Fail()
	}

	stat = fi.Sys().(*syscall.Stat_t)

	if fi.Name() != "dir" {
		t.Fail()
	}

	if fi.Size() != 0 {
		t.Fail()
	}

	if !fi.IsDir() {
		t.Fail()
	}

	if helpers.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if helpers.CurrentGid() != stat.Gid {
		t.Fail()
	}

	if stat.Size != 0 {
		t.Fail()
	}
}

func (t *TestSetup) Setup() error {
	t.MountConfig.DisableWritebackCaching = true

	t.Server = memfs.NewMemFS(helpers.CurrentUid(), helpers.CurrentGid())

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

func (t *TestSetup) TearDown() {
	err := t.destroy()
	if err != nil {
		panic(err)
	}
}

func (t *TestSetup) destroy() (err error) {
	for _, c := range t.ToClose {
		if c == nil {
			continue
		}
		err = c.Close()
		if err != nil {
			return err
		}
	}

	if t.mfs == nil {
		return nil
	}

	if err := unmount(t.Dir); err != nil {
		return fmt.Errorf("unmount: %v", err)
	}

	if err := os.Remove(t.Dir); err != nil {
		return fmt.Errorf("Unlinking mount point: %v", err)
	}

	if err := t.mfs.Join(t.Ctx); err != nil {
		return fmt.Errorf("mfs.Join: %v", err)
	}

	return nil
}

func cleanup() {
	os.RemoveAll("home/jakobwaibel/mountpoint")
}

func unmount(dir string) error {
	delay := 10 * time.Millisecond
	for {
		err := fuse.Unmount(dir)
		if err == nil {
			return err
		}

		if strings.Contains(err.Error(), "resource busy") {
			log.Println("Resource busy! Error while unmounting; trying again")
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}
	}
}
