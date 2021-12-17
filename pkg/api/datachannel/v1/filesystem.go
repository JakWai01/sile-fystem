package api

import (
	"os"
	"time"

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
}

type Dirent struct {
	Offset fuseops.DirOffset
	Inode  fuseops.InodeID
	Name   string
	Type   fuseutil.DirentType
}

type InodeAttributes struct {
	Size   uint64
	Nlink  uint32
	Mode   os.FileMode
	Atime  time.Time
	Mtime  time.Time
	Ctime  time.Time
	Crtime time.Time
	Uid    uint32
	Gid    uint32
}
