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
	// This needs to be replaced
	contents []byte
}

func newInode(name string, path string, attrs fuseops.InodeAttributes) *inode {
	now := time.Now()
	attrs.Mtime = now
	attrs.Crtime = now

	return &inode{
		name:  name,
		path:  path,
		attrs: attrs,
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

func (in *inode) ReadDir(p []byte, offset int) int {
	if !in.isDir() {
		panic("ReadDir called on non-directory.")
	}

	var n int
	for i := offset; i < len(in.entries); i++ {
		e := in.entries[i]

		// Skip the unused entries
		if e.Type == fuseutil.DT_Unknown {
			continue
		}

		tmp := fuseutil.WriteDirent(p[n:], in.entries[i])
		if tmp == 0 {
			break
		}

		n += tmp
	}

	return n
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

func (in *inode) WriteAt(p []byte, off int64) (int, error) {
	if !in.isFile() {
		panic("WriteAt called on non-file.")
	}

	// Update the modification time.
	in.attrs.Mtime = time.Now()

	// Ensure that the contents slice is long enough.
	newLen := int(off) + len(p)
	if len(in.contents) < newLen {
		padding := make([]byte, newLen-len(in.contents))
		in.contents = append(in.contents, padding...)
		in.attrs.Size = uint64(newLen)
	}

	// Copy in the data.
	n := copy(in.contents[off:], p)

	// Sanity check.
	if n != len(p) {
		panic(fmt.Sprintf("Unexpected short copy: %v", n))
	}

	return n, nil
}

func (in *inode) findChild(name string) (i int, ok bool) {
	if !in.isDir() {
		panic("findChild called on non-directory")
	}

	var e fuseutil.Dirent
	for i, e = range in.entries {
		fmt.Println(e.Name)
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
