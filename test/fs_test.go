package filesystem

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/JakWai01/sile-fystem/internal/logging"
	internal "github.com/JakWai01/sile-fystem/internal/test"
	"github.com/JakWai01/sile-fystem/pkg/posix"
)

var (
	verbosity = flag.Int("verbosity", 2, "Verbosity of the logging output")
)

func TestFileSystemSetup(t *testing.T) {
	test := internal.TestSetup{}

	l := logging.NewJSONLogger(*verbosity)

	err := test.Setup(l, true)
	if err != nil {
		t.Fail()
	}

	test = internal.TestSetup{}

	l = logging.NewJSONLogger(*verbosity)

	err = test.Setup(l, true)
	if err != nil {
		t.Fail()
	}
}

func testMkDirOneLevel(test *internal.TestSetup, t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	dirName := path.Join(test.Dir, "dir")

	err = os.Mkdir(dirName, os.ModePerm)
	if err != nil {
		t.Fail()
	}

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

	if posix.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if posix.CurrentGid() != stat.Gid {
		t.Fail()
	}

	if stat.Size != 0 {
		t.Fail()
	}
}

func TestMkdirOneLevel(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testMkDirOneLevel(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testMkDirOneLevel(testMemMapFs, t)
}

func testMkdirTwoLevels(test *internal.TestSetup, t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	err = os.Mkdir(path.Join(test.Dir, "parent"), os.ModePerm)
	if err != nil {
		t.Fail()
	}

	err = os.Mkdir(path.Join(test.Dir, "parent/dir"), os.ModePerm)
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

	if posix.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if posix.CurrentGid() != stat.Gid {
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

func TestMkdirTwoLevels(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testMkdirTwoLevels(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testMkdirTwoLevels(testMemMapFs, t)
}

func testMkdirIntermediateIsFile(test *internal.TestSetup, t *testing.T) {
	var err error

	fileName := path.Join(test.Dir, "foo")

	err = ioutil.WriteFile(fileName, []byte{}, os.ModePerm)
	if err != nil {
		t.Fail()
	}

	dirName := path.Join(fileName, "dir")

	err = os.Mkdir(dirName, os.ModePerm)
	if err == nil {
		t.Fail()
	}

	if !strings.Contains(err.Error(), "not a directory") {
		t.Fail()
	}
}

func TestMkdirIntermediateIsFile(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testMkdirIntermediateIsFile(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testMkdirIntermediateIsFile(testMemMapFs, t)
}

func testMkdirIntermediateIsNonExistent(test *internal.TestSetup, t *testing.T) {
	var err error

	dirName := path.Join(test.Dir, "foo/dir")

	err = os.Mkdir(dirName, os.ModePerm)
	if err == nil {
		t.Fail()
	}

	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fail()
	}
}

func TestMkdirIntermediateIsNonExistent(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testMkdirIntermediateIsNonExistent(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testMkdirIntermediateIsNonExistent(testMemMapFs, t)
}

func testCreateNewFileInRoot(test *internal.TestSetup, t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	fileName := path.Join(test.Dir, "foo")
	const contents = "Hello\x00world"

	err = ioutil.WriteFile(fileName, []byte(contents), os.ModePerm)
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

	if posix.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if posix.CurrentGid() != stat.Gid {
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

func TestCreateNewFileInRoot(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testCreateNewFileInRoot(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testCreateNewFileInRoot(testMemMapFs, t)
}

func testCreateNewFileInSubDir(test *internal.TestSetup, t *testing.T) {
	var err error
	var fi os.FileInfo
	var stat *syscall.Stat_t

	dirName := path.Join(test.Dir, "dir2")

	err = os.Mkdir(dirName, os.ModePerm)
	if err != nil {
		t.Fail()
	}

	fileName := path.Join(dirName, "foo")
	const contents = "Hello\x00world"

	err = ioutil.WriteFile(fileName, []byte(contents), os.ModePerm)
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

	if posix.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if posix.CurrentGid() != stat.Gid {
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

func TestCreateNewFileInSubDir(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testCreateNewFileInSubDir(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testCreateNewFileInSubDir(testMemMapFs, t)
}

func testModifyExistingFileInRoot(test *internal.TestSetup, t *testing.T) {
	var err error
	var n int
	var fi os.FileInfo
	var stat *syscall.Stat_t

	fileName := path.Join(test.Dir, "foo2")

	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), os.ModePerm)
	if err != nil {
		t.Fail()
	}

	f, err := os.OpenFile(fileName, os.O_WRONLY, os.ModePerm)
	if err != nil {
		t.Fail()
	}

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

	if posix.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if posix.CurrentGid() != stat.Gid {
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

func TestModifyExistingFileInRoot(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testModifyExistingFileInRoot(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testModifyExistingFileInRoot(testMemMapFs, t)
}

func testModifyExistingFileInSubDir(test *internal.TestSetup, t *testing.T) {
	var err error
	var n int
	var fi os.FileInfo
	var stat *syscall.Stat_t

	dirName := path.Join(test.Dir, "dir3")

	err = os.Mkdir(dirName, os.ModePerm)
	if err != nil {
		t.Fail()
	}

	fileName := path.Join(dirName, "foo")

	err = ioutil.WriteFile(fileName, []byte("Hello, world!"), os.ModePerm)
	if err != nil {
		t.Fail()
	}

	f, err := os.OpenFile(fileName, os.O_WRONLY, os.ModePerm)
	if err != nil {
		t.Fail()
	}

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

	if posix.CurrentUid() != stat.Uid {
		t.Fail()
	}

	if posix.CurrentGid() != stat.Gid {
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
	testOsFs := setupTestingEnvironment(true)
	testModifyExistingFileInSubDir(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testModifyExistingFileInSubDir(testMemMapFs, t)
}

func testUnlinkFileNonExistent(test *internal.TestSetup, t *testing.T) {
	err := os.Remove(path.Join(test.Dir, "foo3"))
	if err == nil {
		t.Fail()
	}

	if !strings.Contains(err.Error(), "no such file") {
		t.Fail()
	}
}

func TestUnlinkFileNonExistent(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testUnlinkFileNonExistent(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testUnlinkFileNonExistent(testMemMapFs, t)
}

func testUnlinkFileStillOpen(test *internal.TestSetup, t *testing.T) {
	fileName := path.Join(test.Dir, "foo4")

	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		t.Fail()
	}

	n, err := f.Write([]byte("tux"))
	if err != nil {
		t.Fail()
	}

	if n != 3 {
		t.Fail()
	}

	err = os.Remove(fileName)
	if err != nil {
		t.Fail()
	}

	fi, err := f.Stat()
	if err != nil {
		t.Fail()
	}

	if fi.Size() != 3 {
		t.Fail()
	}

	buf := make([]byte, 1024)
	n, err = f.ReadAt(buf, 0)
	if err != io.EOF {
		t.Fail()
	}

	if n != 3 {
		t.Fail()
	}

	if string(buf[:3]) != "tux" {
		t.Fail()
	}

	n, err = f.Write([]byte("burrito"))
	if err != nil {
		t.Fail()
	}

	if len("burrito") != n {
		t.Fail()
	}

	f.Close()
}

func TestUnlinkFileStillOpen(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testUnlinkFileStillOpen(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testUnlinkFileStillOpen(testMemMapFs, t)
}

func testRmDirNonExistent(test *internal.TestSetup, t *testing.T) {
	err := os.Remove(path.Join(test.Dir, "harry"))
	if err == nil {
		t.Fail()
	}

	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fail()
	}
}

func TestRmdirNonExistent(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testRmDirNonExistent(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testRmDirNonExistent(testMemMapFs, t)
}

func testLargeFile(test *internal.TestSetup, t *testing.T) {
	var err error

	f, err := os.Create(path.Join(test.Dir, "foo7"))
	if err != nil {
		t.Fail()
	}

	const size = 1 << 24
	contents := bytes.Repeat([]byte{0x20}, size)

	_, err = io.Copy(f, bytes.NewReader(contents))
	if err != nil {
		t.Fail()
	}

	contents, err = ioutil.ReadFile(f.Name())
	if err != nil {
		t.Fail()
	}

	if size != len(contents) {
		t.Fail()
	}

	f.Close()
}

func TestLargeFile(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testLargeFile(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testLargeFile(testMemMapFs, t)
}

func testAppendMode(test *internal.TestSetup, t *testing.T) {
	var err error
	var n int
	var off int64
	buf := make([]byte, 1024)

	fileName := path.Join(test.Dir, "foo8")

	err = ioutil.WriteFile(fileName, []byte("Jello, "), os.ModePerm)
	if err != nil {
		t.Fail()
	}

	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err != nil {
		t.Fail()
	}

	off, err = f.Seek(2, 0)
	if err != nil {
		t.Fail()
	}

	if off != 2 {
		t.Fail()
	}

	n, err = f.Write([]byte("world!"))
	if err != nil {
		t.Fail()
	}

	if n != 6 {
		t.Fail()
	}

	off, err = getFileOffset(f)
	if err != nil {
		t.Fail()
	}

	if off != 13 {
		t.Fail()
	}

	n, err = f.ReadAt(buf, 0)
	if err != io.EOF {
		t.Fail()
	}

	if string(buf[:n]) != "Jello, world!" {
		t.Fail()
	}

	f.Close()
}

func TestAppendMode(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testAppendMode(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testAppendMode(testMemMapFs, t)
}

func testChmod(test *internal.TestSetup, t *testing.T) {
	var err error

	fileName := path.Join(test.Dir, "foo9")

	err = ioutil.WriteFile(fileName, []byte(""), os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.Chmod(fileName, 0754)
	if err != nil {
	}

	fi, err := os.Stat(fileName)
	if err != nil {
		panic(err)
	}

	if fi.Mode() != os.FileMode(0754) {
		t.Fail()
	}
}

func TestChmod(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testChmod(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testChmod(testMemMapFs, t)
}

func testRenameWithinDirFile(test *internal.TestSetup, t *testing.T) {
	var err error

	parentPath := path.Join(test.Dir, "parent2")

	err = os.Mkdir(parentPath, os.ModePerm)
	if err != nil {
		t.Fail()
	}

	oldPath := path.Join(parentPath, "foo10")

	err = ioutil.WriteFile(oldPath, []byte("taco"), os.ModePerm)
	if err != nil {
		t.Fail()
	}

	newPath := path.Join(parentPath, "bar10")

	err = os.Rename(oldPath, newPath)
	if err != nil {
		t.Fail()
	}

	_, err = os.Stat(oldPath)
	if !os.IsNotExist(err) {
		t.Fail()
	}

	_, err = ioutil.ReadFile(oldPath)
	if !os.IsNotExist(err) {
		t.Fail()
	}

	fi, err := os.Stat(newPath)
	if err != nil {
		t.Fail()
	}

	if int64(len("taco")) != fi.Size() {
		t.Fail()
	}

}

func TestRenameWithinDirFile(t *testing.T) {
	testOsFs := setupTestingEnvironment(true)
	testRenameWithinDirFile(testOsFs, t)

	testMemMapFs := setupTestingEnvironment(false)
	testRenameWithinDirFile(testMemMapFs, t)
}

func getFileOffset(f *os.File) (offset int64, err error) {
	const relativeToCurrent = 1
	return f.Seek(0, relativeToCurrent)
}

func setupTestingEnvironment(osfs bool) *internal.TestSetup {
	test := internal.TestSetup{}

	l := logging.NewJSONLogger(*verbosity)

	err := test.Setup(l, osfs)
	if err != nil {
		panic(err)
	}

	return &test
}
