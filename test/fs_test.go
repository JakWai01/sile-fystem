package filesystem

import (
	"flag"
	"io/ioutil"
	"os"
	"path"
	"strings"
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

func TestMkdirOneLevel(t *testing.T) {
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

func TestMkdirTwoLevels(t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	err = os.Mkdir(path.Join(test.Dir, "parent"), 0700)
	if err != nil {
		t.Fail()
	}

	err = os.Mkdir(path.Join(test.Dir, "parent/dir"), 0754)
	if err != nil {
		t.Fail()
	}

	fi, err = os.Stat(path.Join(test.Dir, "parent/dir"))
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

	fi, err = os.Stat(path.Join(test.Dir, "parent"))
	if err != nil {
		t.Fail()
	}
}

func TestMkdirIntermediateIsFile(t *testing.T) {
	var err error

	fileName := path.Join(test.Dir, "foo")

	err = ioutil.WriteFile(fileName, []byte{}, 0700)
	if err != nil {
		t.Fail()
	}

	dirName := path.Join(fileName, "dir")

	err = os.Mkdir(dirName, 0754)
	if err == nil {
		t.Fail()
	}

	if !strings.Contains(err.Error(), "not a directory") {
		t.Fail()
	}
}

func TestMkdirIntermediateIsNonExistent(t *testing.T) {
	var err error

	dirName := path.Join(test.Dir, "foo/dir")

	err = os.Mkdir(dirName, 0754)
	if err == nil {
		t.Fail()
	}

	if !strings.Contains(err.Error(), "not a directory") {
		t.Fail()
	}
}

func TestCreateNewFileInRoot(t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	fileName := path.Join(test.Dir, "foo")
	const contents = "Hello\x00world"

	err = ioutil.WriteFile(fileName, []byte(contents), 0400)
	if err != nil {
		t.Fail()
	}

	fi, err = os.Stat(fileName)
	if err != nil {
		t.Fail()
	}

	stat = fi.Sys().(*syscall.Stat_t)

	if fi.Name() != "foo" {
		t.Fail()
	}

	if int(fi.Size()) != len(contents) {
		t.Fail()
	}

	if fi.IsDir() {
		t.Fail()
	}

	if helpers.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if helpers.CurrentGid() != stat.Gid {
		t.Fail()
	}

	if int64(len(contents)) != stat.Size {
		t.Fail()
	}

	slice, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fail()
	}

	if contents != string(slice) {
		t.Fail()
	}
}

func TestCreateNewFileInSubDir(t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	dirName := path.Join(test.Dir, "dir2")

	err = os.Mkdir(dirName, 0700)
	if err != nil {
		t.Fail()
	}

	fileName := path.Join(dirName, "foo")
	const contents = "Hello\x00world"

	err = ioutil.WriteFile(fileName, []byte(contents), 0400)
	if err != nil {
		t.Fail()
	}

	fi, err = os.Stat(fileName)
	if err != nil {
		t.Fail()
	}

	stat = fi.Sys().(*syscall.Stat_t)

	if fi.Name() != "foo" {
		t.Fail()
	}

	if int(fi.Size()) != len(contents) {
		t.Fail()
	}

	if fi.IsDir() {
		t.Fail()
	}

	if helpers.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if helpers.CurrentGid() != stat.Gid {
		t.Fail()
	}

	if int64(len(contents)) != stat.Size {
		t.Fail()
	}

	slice, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	if contents != string(slice) {
		t.Fail()
	}
}

func TestModifyExistingFileInRoot(t *testing.T) {
	var err error
	var n int
	var fi os.FileInfo
	var stat *syscall.Stat_t

	// Write a file
	fileName := path.Join(test.Dir, "foo2")

	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	if err != nil {
		t.Fail()
	}

	f, err := os.OpenFile(fileName, os.O_WRONLY, 0400)
	if err != nil {
		t.Fail()
	}

	test.ToClose = append(test.ToClose, f)

	n, err = f.WriteAt([]byte("H"), 0)
	if err != nil {
		t.Fail()
	}

	if n != 1 {
		t.Fail()
	}

	fi, err = os.Stat(fileName)
	if err != nil {
		t.Fail()
	}

	stat = fi.Sys().(*syscall.Stat_t)

	if fi.Name() != "foo2" {
		t.Fail()
	}

	if int64(len("Hello, World!")) != fi.Size() {
		t.Fail()
	}

	if fi.IsDir() {
		t.Fail()
	}

	if helpers.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if helpers.CurrentGid() != stat.Gid {
		t.Fail()
	}

	if int64(len("Hello, world!")) != stat.Size {
		t.Fail()
	}

	slice, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fail()
	}

	if "Hello, world!" != string(slice) {
		t.Fail()
	}

	f.Close()
}

func TestModifyExistingFileInSubDir(t *testing.T) {
	var err error
	var n int
	var fi os.FileInfo
	var stat *syscall.Stat_t

	dirName := path.Join(test.Dir, "dir3")

	err = os.Mkdir(dirName, 0700)
	if err != nil {
		t.Fail()
	}

	fileName := path.Join(dirName, "foo")

	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), 0600)
	if err != nil {
		t.Fail()
	}

	f, err := os.OpenFile(fileName, os.O_WRONLY, 0400)
	if err != nil {
		t.Fail()
	}

	test.ToClose = append(test.ToClose, f)

	n, err = f.WriteAt([]byte("H"), 0)
	if err != nil {
		t.Fail()
	}

	if n != 1 {
		t.Fail()
	}

	fi, err = os.Stat(fileName)
	if err != nil {
		t.Fail()
	}

	stat = fi.Sys().(*syscall.Stat_t)

	if fi.Name() != "foo" {
		t.Fail()
	}

	if int64(len("Hello, world!")) != fi.Size() {
		t.Fail()
	}

	if fi.IsDir() {
		t.Fail()
	}

	if helpers.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if helpers.CurrentGid() != stat.Gid {
		t.Fail()
	}

	if int64(len("Hello, world!")) != stat.Size {
		t.Fail()
	}

	slice, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fail()
	}

	if "Hello, world!" != string(slice) {
		t.Fail()
	}

	f.Close()
}

func TestUnlinkFile_NonExistent(t *testing.T) {
	err := os.Remove(path.Join(test.Dir, "foo3"))
	if err == nil {
		t.Fail()
	}

	if !strings.Contains(err.Error(), "no such file") {
		t.Fail()
	}
}
