package filesystem

import (
	"flag"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/JakWai01/sile-fystem/internal/logging"
	internal "github.com/JakWai01/sile-fystem/internal/test"
	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/helpers"
	"github.com/spf13/afero"
)

const (
	fileMode = 0754
	timeSlop = 25 * time.Millisecond
)

var (
	test      internal.TestSetup
	verbosity = flag.Int("verbosity", 2, "Verbosity of the logging output")
)

func TestFileSystemSetup(t *testing.T) {
	test = internal.TestSetup{}

	l := logging.NewJSONLogger(*verbosity)

	err := test.Setup(filesystem.NewFileSystem(helpers.CurrentUid(), helpers.CurrentGid(), test.Dir, "/", l, afero.NewMemMapFs()))
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
