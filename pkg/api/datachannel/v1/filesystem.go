package api

import (
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

type Inode struct {
	Name     string
	Attrs    fuseops.InodeAttributes
	Entries  []fuseutil.Dirent
	Contents []byte
	Target   string
	Xattrs   map[string][]byte
	ChildID  fuseops.InodeID
}
