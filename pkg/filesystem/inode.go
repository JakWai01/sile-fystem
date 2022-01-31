package filesystem

import (
	"fmt"
	"os"
	"sync"
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
	mu      sync.Mutex
}

func newInode(id fuseops.InodeID, name string, path string, attrs fuseops.InodeAttributes) *inode {
	now := time.Now()
	attrs.Mtime = now
	attrs.Crtime = now

	return &inode{
		id:    id,
		name:  name,
		path:  path,
		attrs: attrs,
	}
}

func (in *inode) isDir() bool {
	return in.attrs.Mode&os.ModeDir != 0
}

func (in *inode) addChild(id fuseops.InodeID, name string, dt fuseutil.DirentType) {
	var index int

	in.attrs.Mtime = time.Now()

	defer func() {
		in.entries[index].Offset = fuseops.DirOffset(index + 1)
	}()

	e := fuseutil.Dirent{
		Inode: id,
		Name:  name,
		Type:  dt,
	}

	for index = range in.entries {
		if in.entries[index].Type == fuseutil.DT_Unknown {
			in.entries[index] = e
			return
		}
	}

	index = len(in.entries)
	in.entries = append(in.entries, e)
}

func (in *inode) removeChild(name string) {
	in.mu.Lock()
	defer in.mu.Unlock()

	in.attrs.Mtime = time.Now()

	i, ok := in.findChild(name)
	if !ok {
		panic(fmt.Sprintf("Unknown child: %s", name))
	}

	in.entries[i] = fuseutil.Dirent{
		Type:   fuseutil.DT_Unknown,
		Offset: fuseops.DirOffset(i + 1),
	}
}

func (in *inode) lookUpChild(name string) (id fuseops.InodeID, typ fuseutil.DirentType, ok bool) {
	in.mu.Lock()
	defer in.mu.Unlock()

	index, ok := in.findChild(name)
	if ok {
		id = in.entries[index].Inode
		typ = in.entries[index].Type
	}

	return id, typ, ok
}

func (in *inode) findChild(name string) (i int, ok bool) {
	if !in.isDir() {
		panic("findChild called on non-directory")
	}

	var e fuseutil.Dirent
	for i, e = range in.entries {
		if e.Name == name {
			return i, true
		}
	}

	return 0, false
}
