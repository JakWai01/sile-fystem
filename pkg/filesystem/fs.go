package filesystem

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/JakWai01/sile-fystem/internal/logging"
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

	log *logging.JSONLogger
}

func NewFileSystem(uid uint32, gid uint32, mountpoint string, root string, logger *logging.JSONLogger, backend afero.Fs) fuse.Server {
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

	fs.inodes[fuseops.RootInodeID] = newInode(fuseops.RootInodeID, mountpoint, root, rootAttrs)

	return fuseutil.NewFileSystemServer(fs)
}

func (fs *fileSystem) getInodeOrDie(id fuseops.InodeID) *inode {
	fs.log.Trace("FUSE.getInodeOrDie", map[string]interface{}{
		"id": id,
	})

	inode := fs.inodes[id]
	if inode == nil {
		panic(fmt.Sprintf("Unknown inode: %v", id))
	}

	return inode
}

func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	fs.log.Debug("FUSE.LookUpInode", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"entry":     op.Entry,
		"OpContext": op.OpContext,
	})

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(parent.path)
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
	fs.log.Debug("FUSE.GetInodeAttributes", map[string]interface{}{
		"inode":                op.Inode,
		"attributes":           op.Attributes,
		"attributesExpiration": op.AttributesExpiration,
		"opContext":            op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(inode.path)
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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	var err error
	if op.Size != nil && op.Handle == nil && *op.Size != 0 {
		// require that truncate to non-zero has to be ftruncate()
		// but allow open(O_TRUNC)
		err = syscall.EBADF
	}

	// inode := fs.getInodeOrDie(op.Inode)

	// Send updated Mode to afero
	// err = fs.backend.Chmod(sanitize(inode.path), *op.Mode)
	// if err != nil {
	// 	panic(err)
	// }

	// // Send updated Uid and Gid to afero
	// err = fs.backend.Chown(sanitize(inode.path), int(op.Attributes.Uid), int(op.Attributes.Gid))
	// if err != nil {
	// 	panic(err)
	// }

	// // Send updated Atime and Mtime to afero
	// fs.backend.Chtimes(sanitize(inode.path), *op.Atime, *op.Mtime)
	// if err != nil {
	// 	panic(err)
	// }

	op.Size = &op.Attributes.Size
	op.Mode = &op.Attributes.Mode
	op.Atime = &op.Attributes.Atime
	op.Mtime = &op.Attributes.Mtime

	op.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)

	return err
}

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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(parent.path)
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

	fs.inodes[hash(newPath)] = newInode(hash(newPath), op.Name, newPath, attrs)

	fs.getInodeOrDie(op.Parent).AddChild(hash(newPath), op.Name, fuseutil.DT_Directory)

	op.Entry.Child = hash(newPath)
	op.Entry.Attributes = attrs
	op.Entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	op.Entry.EntryExpiration = op.Entry.AttributesExpiration

	return nil
}

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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(parent.path)
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

	fs.inodes[hash(newPath)] = newInode(hash(newPath), op.Name, newPath, attrs)

	var entry fuseops.ChildInodeEntry

	entry.Child = hash(newPath)

	entry.Attributes = attrs
	entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	entry.EntryExpiration = entry.AttributesExpiration

	op.Entry = entry

	return nil
}

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
	defer fs.mu.Unlock()

	parent := fs.getInodeOrDie(op.Parent)

	file, err := fs.backend.Open(parent.path)
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

	fs.inodes[hash(newPath)] = newInode(hash(newPath), op.Name, newPath, attrs)

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

	inode := fs.getInodeOrDie(childID)

	inode.path = newPath
	inode.name = op.NewName
	inode.id = hash(newPath)

	newParent.AddChild(hash(newPath), op.NewName, childType)

	oldParent.RemoveChild(op.OldName)

	return nil
}

func (fs *fileSystem) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	fs.log.Debug("FUSE.RmDir", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	parent := fs.getInodeOrDie(op.Parent)

	fs.backend.Remove(op.Name)

	childID, _, ok := parent.LookUpChild(op.Name)
	if !ok {
		return fuse.ENOENT
	}

	child := fs.getInodeOrDie(childID)

	file, err := fs.backend.Open(op.Name)
	if err != nil {
		panic(err)
	}

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if info.Size() != 0 {
		return fuse.ENOTEMPTY
	}

	parent.RemoveChild(op.Name)

	child.attrs.Nlink--

	return nil
}

func (fs *fileSystem) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	fs.log.Debug("FUSE.OpenDir", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(inode.path)
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
	fs.log.Debug("FUSE.ReadDir", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"offset":    op.Offset,
		"dst":       op.Dst,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(inode.path)
	if err != nil {
		panic(err)
	}

	children, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	if !inode.isDir() {
		panic("ReadDir called on  non-directory.")
	}

	var n int
	for i := int(op.Offset); i < len(children); i++ {
		var typ fuseutil.DirentType

		// This limits the DirentType to only directories and files.
		// If additional types are needed, consider adding a type field to the inode.
		if children[i].IsDir() {
			typ = fuseutil.DT_Directory
		} else {
			typ = fuseutil.DT_File
		}

		entry := fuseutil.Dirent{
			Offset: fuseops.DirOffset(i + 1),
			Inode:  hash(concatPath(inode.path, children[i].Name())),
			Name:   children[i].Name(),
			Type:   typ,
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

func (fs *fileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	fs.log.Debug("FUSE.OpenFile", map[string]interface{}{
		"inode":         op.Inode,
		"handle":        op.Handle,
		"keepPageCache": op.KeepPageCache,
		"useDirectID":   op.UseDirectIO,
		"opContext":     op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(inode.path)
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
	fs.log.Debug("FUSE.ReadFile", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"offset":    op.Offset,
		"dst":       op.Dst,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	if op.OpContext.Pid == 0 {
		return fuse.EINVAL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode := fs.getInodeOrDie(op.Inode)

	file, err := fs.backend.Open(inode.path)
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
	fs.log.Debug("FUSE.WriteFile", map[string]interface{}{
		"inode":     op.Inode,
		"handle":    op.Handle,
		"offset":    op.Offset,
		"data":      op.Data,
		"opContext": op.OpContext,
	})

	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode := fs.getInodeOrDie(op.Inode)
	fmt.Printf("Inode with id: %v, name: %v, path: %v, attrs: %v, entries: %v, contents: %v", inode.id, inode.name, inode.path, inode.attrs, inode.entries, inode.contents)
	fmt.Println()

	file, err := fs.backend.OpenFile(inode.path, os.O_RDWR, 7777)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.WriteAt(op.Data, op.Offset)
	if err != nil {
		panic(err)
	}

	inode.attrs.Mtime = time.Now()

	return nil
}

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

	fs.mu.Lock()
	defer fs.mu.Unlock()

	parent := fs.getInodeOrDie(op.Parent)

	_, _, exists := parent.LookUpChild(op.Name)
	if exists {
		return fuse.EEXIST
	}

	target := fs.getInodeOrDie(op.Target)

	now := time.Now()
	target.attrs.Nlink++
	target.attrs.Ctime = now

	parent.AddChild(op.Target, op.Name, fuseutil.DT_File)

	op.Entry.Child = op.Target
	op.Entry.Attributes = target.attrs
	op.Entry.AttributesExpiration = time.Now().Add(365 * 24 * time.Hour)
	op.Entry.EntryExpiration = op.Entry.AttributesExpiration

	return nil
}
func (fs *fileSystem) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	fs.log.Debug("FUSE.Unlink", map[string]interface{}{
		"parent":    op.Parent,
		"name":      op.Name,
		"opContext": op.OpContext,
	})

	return nil
}

func (fs *fileSystem) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	fs.log.Debug("FUSE.ReadSymlink", map[string]interface{}{
		"inode":     op.Inode,
		"target":    op.Target,
		"opContext": op.OpContext,
	})

	return nil
}

func (fs *fileSystem) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	fs.log.Debug("FUSE.GetXattr", map[string]interface{}{
		"inode":     op.Inode,
		"name":      op.Name,
		"dst":       op.Dst,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	return nil
}

func (fs *fileSystem) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	fs.log.Debug("FUSE.ListXattr", map[string]interface{}{
		"inode":     op.Inode,
		"dst":       op.Dst,
		"bytesRead": op.BytesRead,
		"opContext": op.OpContext,
	})

	return nil
}

func (fs *fileSystem) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	fs.log.Debug("FUSE.RemoveXattr", map[string]interface{}{
		"inode":     op.Inode,
		"name":      op.Name,
		"opContext": op.OpContext,
	})

	return nil
}

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

// TODO: Update this function accordingly
func (fs *fileSystem) buildIndex(root string) error {
	fs.log.Trace("FUSE.buildIndex", map[string]interface{}{
		"root": root,
	})

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
	fs.log.Trace("FUSE.getFullyQualifiedPath", map[string]interface{}{
		"id": id,
	})

	path := fs.inodes[id].path

	if path == "" && id != 1 {
		panic(fmt.Sprintf("No inode using id: %v found!", id))
	}

	return path
}

// Sanitize path by removing leading slashes.
func sanitize(path string) string {
	if len(path) > 0 {
		if path[0] == '/' && path[1] == '/' {
			return path[1:]
		}
	}
	return path
}

// Returns the concatenated path sanitized
func concatPath(parentPath string, childName string) string {
	return sanitize(parentPath + "/" + childName)
}
