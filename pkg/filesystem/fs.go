package filesystem

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"syscall"
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

// Do we have persistence at some point? How does a node know that it has some children?

// Compare with "normal" inode implementation
// Maybe start introduce the concept of inodes again

// We probably need to enter actual paths in afero for it to create a file as a child entry

// We need the fully qualified path again
// Create new Inode struct as in the old implementation and then work with these informations

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
	// fs.buildIndex("")

	// The rootnode requires ID 1
	fs.inodes[1] = ""

	fmt.Println(fs.inodes)

	return fuseutil.NewFileSystemServer(fs)
}

// Looks for op.Name in op.Parent
func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	fmt.Println("LookUpInode")
	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.inodes[op.Parent]) > 0 {
		if fs.inodes[op.Parent][0] == '/' {
			optimizedPath = fs.inodes[op.Parent][1:]
			fmt.Println(optimizedPath)
		} else {
			optimizedPath = fs.inodes[op.Parent]
		}
	}

	file, err = fs.backend.Open(optimizedPath)
	if err != nil {
		panic(err)
	}

	children, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	// We have one child ("test"), but the kernel only searches for .Trash and .Trash-1000
	// Try to ignore this but make the kernel search for  "test"
	for _, child := range children {
		fmt.Printf("Child.Name(): %v, op.Name: %v\n", child.Name(), op.Name)
		if child.Name() == op.Name {
			fmt.Println("Found the child")
			op.Entry.Child = hash(child.Name())
			op.Entry.Attributes = fuseops.InodeAttributes{
				Size:   uint64(child.Size()),
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
	fmt.Printf("GetInodeAttributes with op.Inode %v", op.Inode)
	fmt.Println()

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.inodes[op.Inode]) > 0 {
		if fs.inodes[op.Inode][0] == '/' {
			optimizedPath = fs.inodes[op.Inode][1:]
			fmt.Println(optimizedPath)
		} else {
			optimizedPath = fs.inodes[op.Inode]
		}
	}

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
		Size:   uint64(info.Size()),
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

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var err error
	if op.Size != nil && op.Handle == nil && *op.Size != 0 {
		// require that truncate to non-zero has to be ftruncate()
		// but allow open(O_TRUNC)
		err = syscall.EBADF
	}

	op.Size = &op.Attributes.Size
	op.Mode = &op.Attributes.Mode
	op.Atime = &op.Attributes.Atime
	op.Mtime = &op.Attributes.Mtime

	op.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)

	return err
}

// We should let the kernel know about "test"
func (fs *fileSystem) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	fmt.Println("MkDir")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.inodes[op.Parent]) > 0 {
		if fs.inodes[op.Parent][0] == '/' {
			optimizedPath = fs.inodes[op.Parent][1:]
			fmt.Println(optimizedPath)
		} else {
			optimizedPath = fs.inodes[op.Parent]
		}
	}

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

	// Here, we probably need the real path
	err = fs.backend.Mkdir(op.Name, op.Mode)
	if err != nil {
		panic(err)
	}

	fs.inodes[hash(op.Name)] = op.Name

	fmt.Printf("op.Name %v", op.Name)
	fmt.Println()

	op.Entry.Child = hash(op.Name)
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

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var file afero.File
	var err error

	var optimizedPath string

	if len(fs.inodes[op.Parent]) > 0 {
		if fs.inodes[op.Parent][0] == '/' {
			optimizedPath = fs.inodes[op.Parent][1:]
			fmt.Println(optimizedPath)
		} else {
			optimizedPath = fs.inodes[op.Parent]
		}
	}

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

	_, err = fs.backend.Create(op.Name)
	if err != nil {
		panic(err)
	}

	fs.inodes[hash(op.Name)] = op.Name

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(op.Name)

	now := time.Now()
	entry.Attributes = fuseops.InodeAttributes{
		Nlink:  1,
		Mode:   op.Mode,
		Atime:  now,
		Mtime:  now,
		Ctime:  now,
		Crtime: now,
		Uid:    fs.uid,
		Gid:    fs.gid,
	}

	entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	entry.EntryExpiration = entry.AttributesExpiration

	op.Entry = entry

	return nil
}

func (fs *fileSystem) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) (err error) {
	fmt.Println("CreateFile")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var file afero.File

	var optimizedPath string

	if len(fs.inodes[op.Parent]) > 0 {
		if fs.inodes[op.Parent][0] == '/' {
			optimizedPath = fs.inodes[op.Parent][1:]
			fmt.Println(optimizedPath)
		} else {
			optimizedPath = fs.inodes[op.Parent]
		}
	}

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

	_, err = fs.backend.Create(op.Name)
	if err != nil {
		panic(err)
	}

	fs.inodes[hash(op.Name)] = op.Name

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(op.Name)

	now := time.Now()
	entry.Attributes = fuseops.InodeAttributes{
		Nlink:  1,
		Mode:   op.Mode,
		Atime:  now,
		Mtime:  now,
		Ctime:  now,
		Crtime: now,
		Uid:    fs.uid,
		Gid:    fs.gid,
	}

	entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	entry.EntryExpiration = entry.AttributesExpiration

	op.Entry = entry
	return nil
}

func (fs *fileSystem) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	fmt.Println("Rename")

	err := fs.backend.Rename(op.OldName, op.NewName)
	if err != nil {
		panic(err)
	}

	return nil
}

func (fs *fileSystem) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	fmt.Println("RmDir")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.backend.Remove(op.Name)

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

	if len(fs.inodes[op.Inode]) > 0 {
		if fs.inodes[op.Inode][0] == '/' {
			optimizedPath = fs.inodes[op.Inode][1:]
			fmt.Println(optimizedPath)
		} else {
			optimizedPath = fs.inodes[op.Inode]
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

// ReadDir lists a directory
func (fs *fileSystem) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	fmt.Println("ReadDir")
	fmt.Printf("Inode: %v, Handle: %v, Offset: %v, BytesRead: %v, OpContext: %v", op.Inode, op.Handle, op.Offset, op.BytesRead, op.OpContext)
	fmt.Println()
	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var file afero.File
	var err error

	// optimizedPath is always ""
	var optimizedPath string

	if len(fs.inodes[op.Inode]) > 0 {
		if fs.inodes[op.Inode][0] == '/' {
			optimizedPath = fs.inodes[op.Inode][1:]
			fmt.Println(optimizedPath)
		} else {
			optimizedPath = fs.inodes[op.Inode]
		}
	}
	fmt.Printf("optimizedPath: %v", optimizedPath)
	fmt.Println()
	// Grab the directory
	file, err = fs.backend.Open(optimizedPath)
	if err != nil {
		panic(err)
	}

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	children, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	// op.BytesRead = inode.ReadDir(op.Dst, int(op.Offset))
	if !info.IsDir() {
		panic("ReadDir called on non-directory.")
	}

	if int(op.Offset) > len(children) {
		return fuse.EIO
	}

	var n int
	for _, entry := range children[op.Offset:] {

		child := fuseutil.Dirent{
			Name:  entry.Name(),
			Inode: hash(entry.Name()),
			Type:  fuseutil.DT_Directory,
		}

		fmt.Printf("Child: %v", child)
		fmt.Println()
		written := fuseutil.WriteDirent(op.Dst[n:], child)
		if written == 0 {
			break
		}

		n += written
	}

	return nil
}

func (fs *fileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	fmt.Println("OpenFile")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	file, err := fs.backend.Open(fs.inodes[op.Inode])
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

	file, err := fs.backend.Open(fs.inodes[op.Inode])
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

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	file, err := fs.backend.Open(fs.inodes[op.Inode])
	if err != nil {
		panic(err)
	}

	_, err = file.WriteAt(op.Data, op.Offset)
	if err != nil {
		panic(err)
	}

	return nil
}

func (fs *fileSystem) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) (err error) {
	fmt.Println("FlushFile")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

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
func (fs *fileSystem) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	fmt.Println("Unlink")
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
