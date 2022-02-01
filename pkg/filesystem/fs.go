package filesystem

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/JakWai01/sile-fystem/pkg/logging"
	"github.com/JakWai01/sile-fystem/pkg/posix"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/spf13/afero"
)

type fileSystem struct {
	inodes  map[fuseops.InodeID]*inode
	root    string
	backend afero.Fs
	fuseutil.NotImplementedFileSystem

	uid uint32
	gid uint32

	mu sync.Mutex
	op sync.Mutex

	log logging.StructuredLogger

	opened afero.File
}

func NewFileSystem(uid uint32, gid uint32, mountpoint string, root string, logger logging.StructuredLogger, backend afero.Fs) fuse.Server {
	fs := &fileSystem{
		inodes:  make(map[fuseops.InodeID]*inode),
		root:    root,
		backend: backend,
		uid:     uid,
		gid:     gid,

		log: logger,
	}

	rootAttrs := fuseops.InodeAttributes{
		Mode: 0700 | os.ModeDir,
		Uid:  uid,
		Gid:  gid,
	}

	fs.buildIndex(root)

	fs.inodes[fuseops.RootInodeID] = newInode(fuseops.RootInodeID, mountpoint, root, rootAttrs)

	return fuseutil.NewFileSystemServer(fs)
}

// Look up a child by name within a parent directory.
// The kernel sends this when resolving user paths to dentry structs, which are then cached.
func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	fs.log.Debug("FUSE.LookUpInode", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"entry":     op.Entry,
		"OpContext": op.OpContext,
	})

	parent := fs.getInodeOrDie(op.Parent)

	childId, _, ok := parent.lookUpChild(op.Name)
	if !ok {
		return fuse.ENOENT
	}

	child := fs.getInodeOrDie(childId)

	op.Entry.Child = childId
	op.Entry.Attributes = child.attrs
	op.Entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	op.Entry.EntryExpiration = op.Entry.AttributesExpiration

	return nil
}

// Refresh the attributes for an inode whose ID was previously returned in a LookUpInodeOp.
// The kernel sends this when the FUSE VFS layer's cache of inode attributes is stale.
// This is controlled by the AttributesExpiration field of ChildInodeEntry, etc.
func (fs *fileSystem) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	fs.log.Debug("FUSE.GetInodeAttributes", map[string]interface{}{
		"inode":                op.Inode,
		"attributes":           op.Attributes,
		"attributesExpiration": op.AttributesExpiration,
		"opContext":            op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	inode := fs.getInodeOrDie(op.Inode)

	op.Attributes = inode.attrs
	op.AttributesExpiration = time.Now().Add(356 * 24 * time.Hour)

	return nil
}

// Change attributes for an inode.
// The kernel sends this for obvious cases like chmod(2), and for less obvious cases like ftrunctate(2).
func (fs *fileSystem) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	fs.log.Debug("FUSE.SetInodeAttributes", map[string]interface{}{
		"inode":                op.Inode,
		"handle":               op.Handle,
		"size":                 op.Size,
		"mode":                 op.Mode,
		"aTime":                op.Atime,
		"mTime":                op.Mtime,
		"attributes":           op.Attributes,
		"attributesExpiration": op.AttributesExpiration,
		"opContext":            op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	var err error
	if op.Size != nil && op.Handle == nil && *op.Size != 0 {
		// require that truncate to non-zero has to be ftruncate()
		// but allow open(O_TRUNC)
		err = syscall.EBADF
	}

	if fs.inodes[op.Inode] == nil {
		fmt.Println("Not inside")
		return fuse.EEXIST
	}

	inode := fs.getInodeOrDie(op.Inode)

	if op.Mode != nil {
		err = fs.backend.Chmod(inode.path, *op.Mode)
		if err != nil {
			panic(err)
		}
		op.Attributes.Mode = *op.Mode
	}

	if op.Atime != nil && op.Mtime != nil {
		err = fs.backend.Chtimes(inode.path, op.Attributes.Atime, op.Attributes.Mtime)
		if err != nil {
			panic(err)
		}
		op.Attributes.Atime = *op.Atime
		op.Attributes.Mtime = *op.Mtime
	}

	if op.Size != nil {
		op.Attributes.Size = *op.Size
	}

	op.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)

	return err
}

// Create a directory inode as a child of an existing directory inode.
// The kernel sends this in response to a mkdir(2) call.
func (fs *fileSystem) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	fs.log.Debug("FUSE.MkDir", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"mode":      op.Mode,
		"entry":     op.Entry,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	parent := fs.getInodeOrDie(op.Parent)

	_, _, ok := parent.lookUpChild(op.Name)
	if ok {
		return fuse.EEXIST
	}

	newPath := concatPath(parent.path, op.Name)

	err := fs.backend.Mkdir(newPath, op.Mode)
	if err != nil {
		panic(err)
	}

	attrs := fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  op.Mode,
		Uid:   fs.uid,
		Gid:   fs.gid,
	}

	fs.inodes[hash(newPath)] = newInode(hash(newPath), op.Name, newPath, attrs)
	fs.getInodeOrDie(op.Parent).addChild(hash(newPath), op.Name, fuseutil.DT_Directory)

	op.Entry.Child = hash(newPath)
	op.Entry.Attributes = attrs
	op.Entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	op.Entry.EntryExpiration = op.Entry.AttributesExpiration

	return nil
}

// Create a file inode as a child of an existing directory inode. The kernel sends this in response to a mknod(2) call.
func (fs *fileSystem) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	fs.log.Debug("FUSE.MkNode", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"mode":      op.Mode,
		"entry":     op.Entry,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	parent := fs.getInodeOrDie(op.Parent)

	_, _, ok := parent.lookUpChild(op.Name)
	if ok {
		return fuse.EEXIST
	}

	newPath := concatPath(parent.path, op.Name)

	_, err := fs.backend.Create(newPath)
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

	fs.inodes[hash(newPath)] = newInode(hash(newPath), op.Name, newPath, attrs)
	parent.addChild(hash(newPath), op.Name, fuseutil.DT_File)

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(newPath)

	entry.Attributes = attrs
	entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	entry.EntryExpiration = entry.AttributesExpiration

	op.Entry = entry

	return nil
}

// Create a file inode and open it.
// The kernel sends this when the user asks to open a file with the O_CREAT flag and the kernel
// has observed that the file doesn't exist.
func (fs *fileSystem) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) (err error) {
	fs.log.Debug("FUSE.CreateFile", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"mode":      op.Mode,
		"entry":     op.Entry,
		"handle":    op.Handle,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	parent := fs.getInodeOrDie(op.Parent)

	_, _, ok := parent.lookUpChild(op.Name)
	if ok {
		return fuse.EEXIST
	}

	newPath := concatPath(parent.path, op.Name)

	fs.opened, err = fs.backend.Create(newPath)
	if err != nil {
		log.Println(newPath)

		panic(err)
	}

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

	fs.inodes[hash(newPath)] = newInode(hash(newPath), op.Name, newPath, attrs)
	log.Println(op.Parent)
	fs.getInodeOrDie(op.Parent).addChild(hash(newPath), op.Name, fuseutil.DT_File)

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(newPath)

	entry.Attributes = attrs
	entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	entry.EntryExpiration = entry.AttributesExpiration

	op.Entry = entry

	return nil
}

// Rename a file or directory, given the IDs of the original parent directory and the new one (which may be the same).
func (fs *fileSystem) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	fs.log.Debug("FUSE.Rename", map[string]interface{}{
		"oldParent": op.OldParent,
		"oldName":   op.OldName,
		"newParent": op.NewParent,
		"newName":   op.NewName,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	oldParent := fs.getInodeOrDie(op.OldParent)
	oldPath := concatPath(oldParent.path, op.OldName)

	newParent := fs.getInodeOrDie(op.NewParent)
	newPath := concatPath(newParent.path, op.NewName)

	err := fs.backend.Rename(oldPath, newPath)
	if err != nil {
		panic(err)
	}

	childID, childType, ok := oldParent.lookUpChild(op.OldName)
	if !ok {
		return fuse.ENOENT
	}

	existingID, _, ok := newParent.lookUpChild(op.NewName)
	if ok {
		existing := fs.getInodeOrDie(existingID)

		if len(existing.entries) > 0 {
			return fuse.ENOTEMPTY
		}

		newParent.removeChild(op.NewName)
	}

	inode := fs.getInodeOrDie(childID)

	inode.path = newPath
	inode.name = op.NewName

	newParent.addChild(childID, op.NewName, childType)
	oldParent.removeChild(op.OldName)

	return nil
}

// Unlink a directory from its parent.
func (fs *fileSystem) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	fs.log.Debug("FUSE.RmDir", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	parent := fs.getInodeOrDie(op.Parent)

	if err := fs.backend.Remove(op.Name); err != nil {
		panic(err)
	}

	childID, _, ok := parent.lookUpChild(op.Name)
	if !ok {
		return fuse.ENOENT
	}

	child := fs.getInodeOrDie(childID)

	if len(child.entries) > 0 {
		return fuse.ENOTEMPTY
	}

	parent.removeChild(op.Name)
	delete(fs.inodes, childID)

	child.attrs.Nlink--

	return nil
}

// Open a directory inode.
// On Linux the kernel sends this when setting up a struct file for a particular inode with type directory,
// usually in response to an open(2) call from a user-space process. On OS X it may not be sent for every open(2)
func (fs *fileSystem) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	fs.log.Debug("FUSE.OpenDir", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"opContext": op.OpContext,
	})

	var err error

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(inode.path)
	if err != nil {
		panic(err)
	}

	fs.op.Lock()
	defer fs.op.Unlock()

	fs.opened = file

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if !info.IsDir() {
		panic("Found non-dir.")
	}

	return nil
}

// Read entries from a directory previously opened with OpenDir.
func (fs *fileSystem) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	fs.log.Debug("FUSE.ReadDir", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"offset":    op.Offset,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	inode := fs.getInodeOrDie(op.Inode)

	if !inode.isDir() {
		panic("ReadDir called on  non-directory.")
	}

	var n int
	for i := int(op.Offset); i < len(inode.entries); i++ {

		entry := fuseutil.Dirent{
			Offset: fuseops.DirOffset(i + 1),
			Inode:  hash(concatPath(inode.path, inode.entries[i].Name)),
			Name:   inode.entries[i].Name,
			Type:   inode.entries[i].Type,
		}

		tmp := fuseutil.WriteDirent(op.Dst[n:], entry)
		if tmp == 0 {
			break
		}

		n += tmp
	}

	op.BytesRead = n

	return nil
}

// Open a file inode.
// On Linux the kernel sends this when setting up a struct file for a particular inode with type file,
// usually in response to an open(2) call from a user-space process. On OS X it may not be sent for every open(2)
func (fs *fileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	fs.log.Debug("FUSE.OpenFile", map[string]interface{}{
		"inode":         op.Inode,
		"handle":        op.Handle,
		"keepPageCache": op.KeepPageCache,
		"useDirectID":   op.UseDirectIO,
		"opContext":     op.OpContext,
	})

	var err error

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.OpenFile(inode.path, os.O_RDWR|os.O_APPEND, inode.attrs.Mode)
	if err != nil {
		return fuse.EEXIST
	}

	fs.op.Lock()
	defer fs.op.Unlock()

	fs.opened = file

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if info.IsDir() {
		panic("Found non-file.")
	}

	return nil
}

// Read data from a file previously opened with CreateFile or OpenFile.
func (fs *fileSystem) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	fs.log.Debug("FUSE.ReadFile", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"offset":    op.Offset,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	var err error

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.op.Lock()
	defer fs.op.Unlock()

	op.BytesRead, err = fs.opened.ReadAt(op.Dst, op.Offset)
	if err == io.EOF {
		return nil
	}

	return err
}

// Write data to a file previously opened with CreateFile or OpenFile.
func (fs *fileSystem) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	fs.log.Debug("FUSE.WriteFile", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"offset":    op.Offset,
		"opContext": op.OpContext,
		"data":      len(op.Data),
	})

	var err error

	inode := fs.getInodeOrDie(op.Inode)

	fs.op.Lock()
	defer fs.op.Unlock()

	_, err = fs.opened.WriteAt(op.Data, op.Offset)
	if err != nil {
		panic(err)
	}

	inode.attrs.Mtime = time.Now()

	return nil
}

// Create a hard link to an inode
func (fs *fileSystem) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	fs.log.Debug("FUSE.CreateLink", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"target":    op.Target,
		"entry":     op.Entry,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	parent := fs.getInodeOrDie(op.Parent)

	_, _, exists := parent.lookUpChild(op.Name)
	if exists {
		return fuse.EEXIST
	}

	target := fs.getInodeOrDie(op.Target)

	now := time.Now()
	target.attrs.Nlink++
	target.attrs.Ctime = now

	parent.addChild(op.Target, op.Name, fuseutil.DT_File)

	op.Entry.Child = op.Target
	op.Entry.Attributes = target.attrs
	op.Entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	op.Entry.EntryExpiration = op.Entry.AttributesExpiration

	return nil
}

// Write data to a file previously opened with CreateFile or OpenFile.
func (fs *fileSystem) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) (err error) {
	fs.log.Debug("FUSE.FlushFile", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	return nil
}

// Create a symlink inode.
func (fs *fileSystem) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	fs.log.Debug("FUSE.CreateSymlink", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"target":    op.Target,
		"entry":     op.Entry,
		"opContext": op.OpContext,
	})

	return nil
}

// Unlink a file or symlink from its parent
func (fs *fileSystem) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	fs.log.Debug("FUSE.Unlink", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"opContext": op.OpContext,
	})

	parent := fs.getInodeOrDie(op.Parent)

	id, _, _ := parent.lookUpChild(op.Name)

	child := fs.getInodeOrDie(id)

	parent.removeChild(child.name)
	delete(fs.inodes, id)
	fmt.Println(id)

	return fs.backend.Remove(child.path)
}

// Read the target of a symlink inode.
func (fs *fileSystem) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	fs.log.Debug("FUSE.ReadSymlink", map[string]interface{}{
		"inode":     op.Inode,
		"target":    op.Target,
		"opContext": op.OpContext,
	})

	return nil
}

// Get an extended attribute.
func (fs *fileSystem) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	fs.log.Debug("FUSE.GetXattr", map[string]interface{}{
		"inode":     op.Inode,
		"name":      op.Name,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	return nil
}

// List all the extended attributes for a file.
func (fs *fileSystem) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	fs.log.Debug("FUSE.ListXattr", map[string]interface{}{
		"inode":     op.Inode,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	return nil
}

// Remove an extended attribute.
func (fs *fileSystem) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	fs.log.Debug("FUSE.RemoveXattr", map[string]interface{}{
		"inode":     op.Inode,
		"name":      op.Name,
		"opContext": op.OpContext,
	})

	return nil
}

// Set an extended attribute.
func (fs *fileSystem) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	fs.log.Debug("FUSE.SetXattr", map[string]interface{}{
		"inode":     op.Inode,
		"name":      op.Name,
		"value":     op.Value,
		"flags":     op.Flags,
		"opContext": op.OpContext,
	})

	return nil
}

func (fs *fileSystem) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	fs.log.Debug("FUSE.Fallocate", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"offset":    op.Offset,
		"length":    op.Length,
		"mode":      op.Mode,
		"opContext": op.OpContext,
	})

	return nil
}

func (fs *fileSystem) ReleaseFileHandle(ctx context.Context, op *fuseops.ReleaseFileHandleOp) error {
	log.Println("Releasing file")

	fs.op.Lock()
	defer fs.op.Unlock()

	if fs.opened != nil {
		if err := fs.opened.Close(); err != nil {
			panic(err)
		}
	}
	fs.opened = nil
	fs.mu.Unlock()

	return nil
}

func (fs *fileSystem) ReleaseDirHandle(ctx context.Context, op *fuseops.ReleaseDirHandleOp) error {
	log.Println("Releasing dir")

	fs.op.Lock()
	defer fs.op.Unlock()

	if fs.opened != nil {
		if err := fs.opened.Close(); err != nil {
			panic(err)
		}
	}
	fs.opened = nil
	fs.mu.Unlock()

	return nil
}

func (fs *fileSystem) buildIndex(root string) error {
	fs.log.Trace("FUSE.buildIndex", map[string]interface{}{
		"root": root,
	})

	file, err := fs.backend.Open(root)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	attrs := fuseops.InodeAttributes{
		Size:   uint64(info.Size()),
		Mode:   info.Mode(),
		Atime:  info.ModTime(),
		Mtime:  info.ModTime(),
		Ctime:  info.ModTime(),
		Crtime: info.ModTime(),
		Uid:    posix.CurrentUid(),
		Gid:    posix.CurrentGid(),
	}

	fs.inodes[hash(root)] = newInode(hash(root), info.Name(), root, attrs)

	if info.IsDir() {
		children, err := file.Readdir(-1)
		if err != nil {
			panic(err)
		}

		for _, child := range children {
			if child.IsDir() {
				fs.getInodeOrDie(hash(root)).addChild(hash(concatPath(root, child.Name())), child.Name(), fuseutil.DT_Directory)
			} else {
				fs.getInodeOrDie(hash(root)).addChild(hash(concatPath(root, child.Name())), child.Name(), fuseutil.DT_File)
			}
			fs.buildIndex(concatPath(root, child.Name()))
		}
	}

	return nil
}

func (fs *fileSystem) getInodeOrDie(id fuseops.InodeID) *inode {
	fs.log.Trace("FUSE.getInodeOrDie", map[string]interface{}{
		"id": id,
	})

	for _, inode := range fs.inodes {
		fs.log.Trace("FUSE.getInodeOrDieInode", map[string]interface{}{
			"id":   inode.id,
			"name": inode.name,
			"path": inode.path,
		})
	}

	inode := fs.inodes[id]
	if inode == nil {
		panic(fmt.Sprintf("Unknown inode: %v", id))
	}

	return inode
}

func sanitize(path string) string {
	if len(path) > 0 {
		if path[0] == '/' && path[1] == '/' {
			return path[1:]
		}
	}
	return path
}

func concatPath(parentPath string, childName string) string {
	return sanitize(parentPath + "/" + childName)
}

func hash(s string) fuseops.InodeID {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fuseops.InodeID(h.Sum64())
}
