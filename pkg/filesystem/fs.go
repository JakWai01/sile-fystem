package filesystem

// Our hashes need to be the paths as well. Not only the name
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

// Do we have persistence at some point? How does a node know that it has some children?

// Compare with "normal" inode implementation
// Maybe start introduce the concept of inodes again

// We probably need to enter actual paths in afero for it to create a file as a child entry

// We need the fully qualified path again
// Create new Inode struct as in the old implementation and then work with these informations

type fileSystem struct {
	inodes map[fuseops.InodeID]*inode
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
		inodes:  make(map[fuseops.InodeID]*inode),
		root:    root,
		backend: afero.NewMemMapFs(),
		uid:     uid,
		gid:     gid,
	}

	// Build index to store fully qualified path of inode and its ID
	// If absolute paths are needed in the map, just pass the root as an argument.
	// fs.buildIndex("")

	// The rootnode requires ID 1
	rootAttrs := fuseops.InodeAttributes{
		Mode: 0700 | os.ModeDir,
		Uid:  uid,
		Gid:  gid,
	}

	// In this case, root gets passed to the name parameter, since the path is the root of the new filesystem.
	fs.inodes[fuseops.RootInodeID] = newInode(root, "", rootAttrs)

	fmt.Println(fs.inodes)

	return fuseutil.NewFileSystemServer(fs)
}

// Return inode by id. Panic if the inode doesn't exist.
func (fs *fileSystem) getInodeOrDie(id fuseops.InodeID) *inode {
	fmt.Println("getInodeOrDie")
	inode := fs.inodes[id]
	if inode == nil {
		panic(fmt.Sprintf("Unknown inode: %v", id))
	}

	return inode
}

// Looks for op.Name in op.Parent
func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	fmt.Println("LookUpInode")
	fmt.Printf("fs.getInodeORDie(op.Parent).path %v, op.Parent %v", fs.getInodeOrDie(op.Parent).path, op.Parent)
	fmt.Println()

	file, err := fs.backend.Open(fs.getInodeOrDie(op.Parent).path)
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
			op.Entry.Child = hash(fs.getInodeOrDie(op.Parent).path + "/" + child.Name())
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

	file, err := fs.backend.Open(fs.getInodeOrDie(op.Inode).path)
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

	file, err := fs.backend.Open(fs.getInodeOrDie(op.Parent).path)
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

	// Use absolute path to create directories
	path := fs.getInodeOrDie(op.Parent).path

	err = fs.backend.Mkdir(path+"/"+op.Name, op.Mode)
	if err != nil {
		panic(err)
	}

	parentPath := fs.inodes[op.Parent].path

	attrs := fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  op.Mode,
		Uid:   fs.uid,
		Gid:   fs.gid,
	}

	fmt.Printf("op.Name %v", op.Name)
	fmt.Println()

	fs.inodes[hash(parentPath+"/"+op.Name)] = newInode(op.Name, parentPath+"/"+op.Name, attrs)

	fs.getInodeOrDie(op.Parent).AddChild(hash(parentPath+"/"+op.Name), op.Name, fuseutil.DT_Directory)

	op.Entry.Child = hash(parentPath + "/" + op.Name)
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

	file, err := fs.backend.Open(fs.getInodeOrDie(op.Parent).path)
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

	path := fs.getInodeOrDie(op.Parent).path

	_, err = fs.backend.Create(path + "/" + op.Name)
	if err != nil {
		panic(err)
	}

	parentPath := fs.inodes[op.Parent].path
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

	fs.inodes[hash(parentPath+"/"+op.Name)] = newInode(op.Name, parentPath+"/"+op.Name, attrs)

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(parentPath + "/" + op.Name)

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

	file, err := fs.backend.Open(fs.getInodeOrDie(op.Parent).path)
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

	// This parent path is empty
	path := fs.getInodeOrDie(op.Parent).path
	fmt.Printf("Parent path %v", path)
	fmt.Println()

	fmt.Printf("fs.backend.Create %v", path+"/"+op.Name)
	fmt.Println()
	_, err = fs.backend.Create(path + "/" + op.Name)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	// These attributes are not the same as the file has since fs.backend.Create does not take attrs
	// If we would stat the file now, we would probably have different attributes
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

	fs.inodes[hash(path+"/"+op.Name)] = newInode(op.Name, path+"/"+op.Name, attrs)

	fs.getInodeOrDie(op.Parent).AddChild(hash(path+"/"+op.Name), op.Name, fuseutil.DT_File)

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(path + "/" + op.Name)

	entry.Attributes = attrs
	entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	entry.EntryExpiration = entry.AttributesExpiration

	op.Entry = entry

	return nil
}

func (fs *fileSystem) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	fmt.Println("Rename")

	// Maybe need to work on the paths here as well
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

	// Can't open this file because we used the absolute Path to create it
	file, err := fs.backend.Open(fs.getInodeOrDie(op.Inode).path)
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

	// Grab the directory
	inode := fs.getInodeOrDie(op.Inode)

	// Serve the request
	op.BytesRead = inode.ReadDir(op.Dst, int(op.Offset))

	return nil
}

func (fs *fileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	fmt.Println("OpenFile")
	fmt.Println(op.Inode)
	// Apparently lol.txt is a directory as info.IsDir() is true.
	fmt.Println(hash("lol.txt"))
	fmt.Printf("Path: %v", fs.inodes[op.Inode].path)
	fmt.Printf("GetInodeOrDie: %v", fs.getInodeOrDie(op.Inode).path)
	fmt.Println()

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	file, err := fs.backend.Open(fs.inodes[op.Inode].path)
	if err != nil {
		panic(err)
	}

	// We are statting the path here, since it is zero, the root will be evaluated
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

	file, err := fs.backend.Open(fs.inodes[op.Inode].path)
	if err != nil {
		panic(err)
	}

	// Serve the request
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

	// This needs to be updated
	afero.WriteFile(fs.backend, fs.inodes[op.Inode].name, op.Data, 0644)

	_, err := fs.backend.Stat(fs.inodes[op.Inode].name)
	if os.IsNotExist(err) {
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

	// Open all files and Stat to create the nodes

	// Write current root to map
	// fs.inodes[hash(root)] = root

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

	path := fs.inodes[id].path

	if path == "" && id != 1 {
		panic(fmt.Sprintf("No inode using id: %v found!", id))
	}

	return path
}
