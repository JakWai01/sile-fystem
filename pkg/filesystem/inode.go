package filesystem

import (
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
