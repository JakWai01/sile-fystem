package filesystem

import (
	"os"
	"time"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

// Try to store the minimum amount necessary to use afero
type inode struct {
	id      fuseops.InodeID
	name    string
	path    string
	attrs   fuseops.InodeAttributes
	entries []fuseutil.Dirent
}

func newInode(name string, path string, attrs fuseops.InodeAttributes) *inode {
	// Update time info
	now := time.Now()
	attrs.Mtime = now
	attrs.Crtime = now

	// Create the object
	return &inode{
		name:  name,
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
