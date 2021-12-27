package filesystem

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
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

type fileSystem struct {
	inodes  map[fuseops.InodeID]*inode
	root    string
	backend afero.Fs
	fuseutil.NotImplementedFileSystem

	mu syncutil.InvariantMutex

	uid uint32
	gid uint32
}

func NewFileSystem(uid uint32, gid uint32, root string) fuse.Server {
	fs := &fileSystem{
		inodes:  make(map[fuseops.InodeID]*inode),
		root:    root,
		backend: afero.NewMemMapFs(),
		uid:     uid,
		gid:     gid,
	}

	rootAttrs := fuseops.InodeAttributes{
		Mode: 0700 | os.ModeDir,
		Uid:  uid,
		Gid:  gid,
	}

	fs.inodes[fuseops.RootInodeID] = newInode(root, sanitize(""), rootAttrs)

	return fuseutil.NewFileSystemServer(fs)
}

func (fs *fileSystem) getInodeOrDie(id fuseops.InodeID) *inode {
	fmt.Println("getInodeOrDie")

	inode := fs.inodes[id]
	if inode == nil {
		panic(fmt.Sprintf("Unknown inode: %v", id))
	}

	return inode
}

func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	fmt.Println("LookUpInode")

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(sanitize(parent.path))
	if err != nil {
		panic(err)
	}

	children, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	for _, child := range children {
		if child.Name() == op.Name {
			op.Entry.Child = hash(concatPath(parent.path, child.Name()))
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

func (fs *fileSystem) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	fmt.Println("GetInodeAttributes")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(sanitize(inode.path))
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

func (fs *fileSystem) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	fmt.Println("MkDir")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(sanitize(parent.path))
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

	newPath := concatPath(parent.path, op.Name)

	err = fs.backend.Mkdir(newPath, op.Mode)
	if err != nil {
		panic(err)
	}

	attrs := fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  op.Mode,
		Uid:   fs.uid,
		Gid:   fs.gid,
	}

	fs.inodes[hash(newPath)] = newInode(op.Name, newPath, attrs)

	fs.getInodeOrDie(op.Parent).AddChild(hash(newPath), op.Name, fuseutil.DT_Directory)

	op.Entry.Child = hash(newPath)
	op.Entry.Attributes = attrs
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

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(sanitize(parent.path))
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

	newPath := concatPath(parent.path, op.Name)

	_, err = fs.backend.Create(newPath)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	attrs := fuseops.InodeAttributes{
		Nlink:  1,
		Mode:   op.Mode,
		Atime:  now,
		Mtime:  now,
		Ctime:  now,
		Crtime: now,
		Uid:    fs.uid,
		Gid:    fs.gid,
	}

	fs.inodes[hash(newPath)] = newInode(op.Name, newPath, attrs)

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(newPath)

	entry.Attributes = attrs
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

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(sanitize(parent.path))
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

	newPath := concatPath(parent.path, op.Name)

	_, err = fs.backend.Create(newPath)
	if err != nil {
		panic(err)
	}

	// Set permissions of file to op.Mode
	err = fs.backend.Chmod(newPath, op.Mode)
	if err != nil {
		panic(err)
	}

	now := time.Now()

	attrs := fuseops.InodeAttributes{
		Nlink:  1,
		Mode:   op.Mode,
		Atime:  now,
		Mtime:  now,
		Ctime:  now,
		Crtime: now,
		Uid:    fs.uid,
		Gid:    fs.gid,
	}

	fs.inodes[hash(newPath)] = newInode(op.Name, newPath, attrs)

	fs.getInodeOrDie(op.Parent).AddChild(hash(newPath), op.Name, fuseutil.DT_File)

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(newPath)

	entry.Attributes = attrs
	entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	entry.EntryExpiration = entry.AttributesExpiration

	op.Entry = entry

	return nil
}

func (fs *fileSystem) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	fmt.Println("Rename")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	oldParent := fs.getInodeOrDie(op.OldParent)
	oldPath := concatPath(oldParent.path, op.OldName)

	newParent := fs.getInodeOrDie(op.NewParent)
	newPath := concatPath(newParent.path, op.NewName)

	err := fs.backend.Rename(oldPath, newPath)
	if err != nil {
		panic(err)
	}

	childID, childType, ok := oldParent.LookUpChild(op.OldName)
	if !ok {
		return fuse.ENOENT
	}

	existingID, _, ok := newParent.LookUpChild(op.NewName)
	if ok {
		existing := fs.getInodeOrDie(existingID)

		var buf [4096]byte
		if existing.isDir() && existing.ReadDir(buf[:], 0) > 0 {
			return fuse.ENOTEMPTY
		}

		newParent.RemoveChild(op.NewName)
	}

	newParent.AddChild(childID, op.NewName, childType)

	oldParent.RemoveChild(op.OldName)

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

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(sanitize(inode.path))
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode := fs.getInodeOrDie(op.Inode)

	op.BytesRead = inode.ReadDir(op.Dst, int(op.Offset))

	return nil
}

func (fs *fileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	fmt.Println("OpenFile")

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(sanitize(inode.path))
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(sanitize(inode.path))
	if err != nil {
		panic(err)
	}

	op.BytesRead, err = file.ReadAt(op.Dst, op.Offset)
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

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.OpenFile(sanitize(inode.path), os.O_WRONLY, 0400)
	if err != nil {
		panic(err)
	}
	defer file.Close()

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

// TODO: Update this function accordingly
func (fs *fileSystem) buildIndex(root string) error {
	fmt.Println("buildIndex")
	fmt.Printf("current root: %v", root)
	fmt.Println()

	// Open all files and Stat to create the nodes

	// Write current root to map
	// fs.inodes[hash(root)] = root

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
			fs.buildIndex(concatPath(root, child.Name()))
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

	path := fs.inodes[id].path

	if path == "" && id != 1 {
		panic(fmt.Sprintf("No inode using id: %v found!", id))
	}

	return path
}

// Sanitize path by removing leading slashes.
func sanitize(path string) string {
	if len(path) > 0 {
		if path[0] == '/' {
			return path[1:]
		}
	}
	return path
}

// Returns the concatenated path sanitized
func concatPath(parentPath string, childName string) string {
	return sanitize(parentPath + "/" + childName)
}
