package filesystem

import (
	"fmt"
	"os"
	"time"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

type inode struct {
	id      fuseops.InodeID
	name    string
	path    string
	attrs   fuseops.InodeAttributes
	entries []fuseutil.Dirent

	onError func(err interface{})
}

func newInode(id fuseops.InodeID, name string, path string, attrs fuseops.InodeAttributes, onError func(err interface{})) *inode {
	now := time.Now()
	attrs.Mtime = now
	attrs.Crtime = now

	return &inode{
		id:    id,
		name:  name,
		path:  path,
		attrs: attrs,

		onError: onError,
	}
}

func (in *inode) isDir() bool {
	return in.attrs.Mode&os.ModeDir != 0
}

func (in *inode) isSymlink() bool {
	return in.attrs.Mode&os.ModeSymlink != 0
}

func (in *inode) isFile() bool {
	return !(in.isDir() || in.isSymlink())
}

func (in *inode) AddChild(id fuseops.InodeID, name string, dt fuseutil.DirentType) {
	var index int

	// Update the modification time
	in.attrs.Mtime = time.Now()

	// No matter where we place the entry, make sure it has the correct Offset field
	defer func() {
		in.entries[index].Offset = fuseops.DirOffset(index + 1)
	}()

	// Set up the entry
	e := fuseutil.Dirent{
		Inode: id,
		Name:  name,
		Type:  dt,
	}

	// Look for a gap in which we can insert it
	for index = range in.entries {
		if in.entries[index].Type == fuseutil.DT_Unknown {
			in.entries[index] = e
			return
		}
	}

	// Append it to the end
	index = len(in.entries)
	in.entries = append(in.entries, e)
}

func (in *inode) RemoveChild(name string) {
	in.attrs.Mtime = time.Now()

	i, ok := in.findChild(name)
	if !ok {
		in.onError(fmt.Sprintf("Unknown child: %s", name))
	}

	in.entries[i] = fuseutil.Dirent{
		Type:   fuseutil.DT_Unknown,
		Offset: fuseops.DirOffset(i + 1),
	}
}

func (in *inode) findChild(name string) (i int, ok bool) {
	if !in.isDir() {
		in.onError("findChild called on non-directory")
	}

	var e fuseutil.Dirent
	for i, e = range in.entries {
		if e.Name == name {
			return i, true
		}
	}

	return 0, false
}

func (in *inode) LookUpChild(name string) (id fuseops.InodeID, typ fuseutil.DirentType, ok bool) {
	index, ok := in.findChild(name)
	if ok {
		id = in.entries[index].Inode
		typ = in.entries[index].Type
	}

	return id, typ, ok
}
