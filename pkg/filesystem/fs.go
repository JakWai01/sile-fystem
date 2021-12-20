package filesystem

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/syncutil"
	"github.com/spf13/afero"
)

type InodeNotFound struct{}

func (e *InodeNotFound) Error() string {
	return "Inode not found in FUSE"
}

type fileSystem struct {
	inodes map[fuseops.InodeID]string
	root   string
	// backend afero.Fs
	backend afero.Fs
	fuseutil.NotImplementedFileSystem

	mu syncutil.InvariantMutex

	uid uint32
	gid uint32
}

func NewFileSystem(uid uint32, gid uint32, root string) fuse.Server {
	fs := &fileSystem{
		inodes:  make(map[fuseops.InodeID]string),
		root:    root,
		backend: afero.NewMemMapFs(),
		uid:     uid,
		gid:     gid,
	}

	// Build index to store fully qualified path of inode and its ID
	// If absolute paths are needed in the map, just pass the root as an argument.
	fs.buildIndex("")

	// The rootnode requires ID 1
	fs.inodes[1] = ""

	fmt.Println(fs.inodes)

	return fuseutil.NewFileSystemServer(fs)
}

// Looks for op.Name in op.Parent
// This function is probably also used to assert an ID to each child on initilization
func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	fmt.Println("LookUpInode")

	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.getFullyQualifiedPath(op.Parent)) > 0 {
		if fs.getFullyQualifiedPath(op.Parent)[0] == '/' {
			optimizedPath = fs.getFullyQualifiedPath(op.Parent)[1:]
			fmt.Println(optimizedPath)
		}
	}

	// Open triggers the GetInodeAttributes function again, which leads to an infinite loop
	file, err = fs.backend.Open(optimizedPath)
	if err != nil {
		panic(err)
	}

	children, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	for _, child := range children {
		fmt.Sprintf("Fully qualified path of parent in LookUpInode: %v", fs.getFullyQualifiedPath(op.Parent))
		childPath := fs.getFullyQualifiedPath(op.Parent) + "/" + child.Name()
		if child.Name() == op.Name {
			op.Entry.Child = hash(childPath)
			op.Entry.Attributes = fuseops.InodeAttributes{
				Size: uint64(child.Size()),
				// TODO
				Nlink:  1,
				Mode:   child.Mode(),
				Atime:  child.ModTime(),
				Mtime:  child.ModTime(),
				Ctime:  child.ModTime(),
				Crtime: child.ModTime(),
				Uid:    fs.uid,
				Gid:    fs.gid,
			}

			op.Entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
			op.Entry.EntryExpiration = op.Entry.AttributesExpiration
		}
	}

	return nil
}

// Get attributes of op.InodeID
func (fs *fileSystem) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	fmt.Printf("GetInodeAttributes with op.Inode: %v and path: %v", op.Inode, fs.getFullyQualifiedPath(op.Inode))
	fmt.Println()

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.getFullyQualifiedPath(op.Inode)) > 0 {
		if fs.getFullyQualifiedPath(op.Inode)[0] == '/' {
			optimizedPath = fs.getFullyQualifiedPath(op.Inode)[1:]
			fmt.Println(optimizedPath)
		}
	}

	// Open triggers the GetInodeAttributes function again, which leads to an infinite loop
	file, err = fs.backend.Open(optimizedPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	op.Attributes = fuseops.InodeAttributes{
		Size: uint64(info.Size()),
		// TODO
		Mode:   info.Mode(),
		Atime:  info.ModTime(),
		Mtime:  info.ModTime(),
		Ctime:  info.ModTime(),
		Crtime: info.ModTime(),
		Uid:    fs.uid,
		Gid:    fs.gid,
	}

	op.AttributesExpiration = time.Now().Add(356 * 24 * time.Hour)

	return nil
}

func (fs *fileSystem) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	fmt.Println("SetInodeAttributes")
	return nil
}

func (fs *fileSystem) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	fmt.Println("MkDir")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	fmt.Sprintf("Parent ID: %v", op.Parent)
	// Grab the parent

	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.getFullyQualifiedPath(op.Parent)) > 0 {
		if fs.getFullyQualifiedPath(op.Parent)[0] == '/' {
			optimizedPath = fs.getFullyQualifiedPath(op.Parent)[1:]
			fmt.Println(optimizedPath)
		}
	}

	// Open triggers the GetInodeAttributes function again, which leads to an infinite loop
	file, err = fs.backend.Open(optimizedPath)
	if err != nil {
		panic(err)
	}

	children, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	for _, child := range children {
		if child.Name() == op.Name {
			return fuse.EEXIST
		}
	}

	err = fs.backend.Mkdir(op.Name, op.Mode)
	if err != nil {
		panic(err)
	}

	fs.inodes[hash(fs.getFullyQualifiedPath(op.Parent)+"/"+op.Name)] = fs.getFullyQualifiedPath(op.Parent) + "/" + op.Name

	op.Entry.Child = hash(fs.getFullyQualifiedPath(op.Parent) + "/" + op.Name)
	op.Entry.Attributes = fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  op.Mode,
		Uid:   fs.uid,
		Gid:   fs.gid,
	}

	op.Entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	op.Entry.EntryExpiration = op.Entry.AttributesExpiration

	return nil
}

func (fs *fileSystem) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	fmt.Println("MkNode")
	return nil
}

func (fs *fileSystem) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) (err error) {
	fmt.Println("CreateFile")
	return nil
}

func (fs *fileSystem) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	fmt.Println("CreateSymlink")
	return nil
}

func (fs *fileSystem) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	fmt.Println("CreateLink")
	return nil
}

func (fs *fileSystem) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	fmt.Println("Rename")
	return nil
}

func (fs *fileSystem) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	fmt.Println("RmDir")
	return nil
}

func (fs *fileSystem) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	fmt.Println("Unlink")
	return nil
}

func (fs *fileSystem) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	fmt.Println("OpenDir")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}
	fmt.Println(fs.inodes)
	fmt.Println(op.Inode)

	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.getFullyQualifiedPath(op.Inode)) > 0 {
		if fs.getFullyQualifiedPath(op.Inode)[0] == '/' {
			optimizedPath = fs.getFullyQualifiedPath(op.Inode)[1:]
			fmt.Println(optimizedPath)
		}
	}

	file, err = fs.backend.Open(optimizedPath)
	if err != nil {
		panic(err)
	}

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if !info.IsDir() {
		panic("Found non-dir.")
	}

	return nil
}

func (fs *fileSystem) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	fmt.Println("ReadDir")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	op.BytesRead = 0

	// file, err := fs.backend.Open(fs.getFullyQualifiedPath(op.Inode))
	// if err != nil {
	// 	panic(err)
	// }

	// TODO: Is this the right function?
	// op.BytesRead, err = file.ReadAt(op.Dst, int64(op.Offset))
	// if err != nil {
	// 	panic(err)
	// }

	return nil
}

func (fs *fileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	fmt.Println("OpenFile")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	file, err := fs.backend.Open(fs.getFullyQualifiedPath(op.Inode))
	if err != nil {
		panic(err)
	}

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if info.IsDir() {
		panic("Found non-file.")
	}

	return nil
}

func (fs *fileSystem) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	fmt.Println("ReadFile")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	file, err := fs.backend.Open(fs.getFullyQualifiedPath(op.Inode))
	if err != nil {
		panic(err)
	}

	// TODO: Is this the right function?
	op.BytesRead, err = file.ReadAt(op.Dst, int64(op.Offset))
	if err == io.EOF {
		return nil
	}

	return err
}

func (fs *fileSystem) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	fmt.Println("WriteFile")
	return nil
}

func (fs *fileSystem) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) (err error) {
	fmt.Println("FlushFile")
	return nil
}

func (fs *fileSystem) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	fmt.Println("ReadSymlink")
	return nil
}

func (fs *fileSystem) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	fmt.Println("GetXattr")
	return nil
}

func (fs *fileSystem) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	fmt.Println("ListXattr")
	return nil
}

func (fs *fileSystem) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	fmt.Println("RemoveXattr")
	return nil
}

func (fs *fileSystem) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	fmt.Println("SetXattr")
	return nil
}

func (fs *fileSystem) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	fmt.Println("Fallocate")
	return nil
}

// Build index using the root node.
func (fs *fileSystem) buildIndex(root string) error {
	fmt.Println("buildIndex")
	fmt.Printf("current root: %v", root)
	fmt.Println()

	// Write current root to map
	fs.inodes[hash(root)] = root

	fmt.Println(root)
	file, err := fs.backend.Open(root)
	if err != nil {
		panic(err)
	}

	children, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	for _, child := range children {
		print(child.Name())
		if child.IsDir() {
			fs.buildIndex(root + "/" + child.Name())
		}
	}

	return nil
}

func hash(s string) fuseops.InodeID {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fuseops.InodeID(h.Sum32())
}

func (fs *fileSystem) getFullyQualifiedPath(id fuseops.InodeID) string {
	fmt.Println("getFullyQualifiedPath")

	path := fs.inodes[id]

	if path == "" && id != 1 {
		panic(fmt.Sprintf("No inode using id: %v found!", id))
	}

	return path
}