package filesystem

import (
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
