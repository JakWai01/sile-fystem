package hellofs

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
)

func NewHelloFS(clock timeutil.Clock) (fuse.Server, error) {
	fs := &helloFS{
		Clock: clock,
	}

	return fuseutil.NewFileSystemServer(fs), nil
}

type helloFS struct {
	fuseutil.NotImplementedFileSystem

	Clock timeutil.Clock
}

const (
	rootInode fuseops.InodeID = fuseops.RootInodeID + iota
	helloInode
	dirInode
	worldInode
)

type inodeInfo struct {
	attributes fuseops.InodeAttributes

	// File or directory
	dir bool

	// For directories, children
	children []fuseutil.Dirent
}

// We have a fixed directory structure
// This is the structure we are working with. It is custom made.
var gInodeInfo = map[fuseops.InodeID]inodeInfo{
	// root
	rootInode: inodeInfo{
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0555 | os.ModeDir,
		},
		dir: true,
		children: []fuseutil.Dirent{
			fuseutil.Dirent{
				Offset: 1,
				Inode:  helloInode,
				Name:   "hello",
				Type:   fuseutil.DT_File,
			},
			fuseutil.Dirent{
				Offset: 2,
				Inode:  dirInode,
				Name:   "dir",
				Type:   fuseutil.DT_Directory,
			},
		},
	},

	// hello
	helloInode: inodeInfo{
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0444,
			Size:  uint64(len("Hello, world!")),
		},
	},

	// dir
	dirInode: inodeInfo{
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0555 | os.ModeDir,
		},
		dir: true,
		children: []fuseutil.Dirent{
			fuseutil.Dirent{
				Offset: 1,
				Inode:  worldInode,
				Name:   "world",
				Type:   fuseutil.DT_File,
			},
		},
	},

	// world
	worldInode: inodeInfo{
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0444,
			Size:  uint64(len("Hello, world!")),
		},
	},
}

func findChildInode(name string, children []fuseutil.Dirent) (fuseops.InodeID, error) {
	for _, e := range children {
		if e.Name == name {
			return e.Inode, nil
		}
	}
	return 0, fuse.ENOENT
}

func (fs *helloFS) patchAttributes(attr *fuseops.InodeAttributes) {
	now := fs.Clock.Now()
	// Time of last access
	attr.Atime = now
	// Time of last modification
	attr.Mtime = now
	// Time of creation (OS X only)
	attr.Crtime = now
}

// StatFS system call is supposed tu return information about a mounted filesytem
func (fs *helloFS) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return nil
}

func (fs *helloFS) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	// Find the info for the parent
	parentInfo, ok := gInodeInfo[op.Parent]
	if !ok {
		return fuse.ENOENT
	}

	// Find the child within the parent
	childInode, err := findChildInode(op.Name, parentInfo.children)
	if err != nil {
		return err
	}

	// Copy over information
	op.Entry.Child = childInode
	op.Entry.Attributes = gInodeInfo[childInode].attributes

	// Path attributes
	fs.patchAttributes(&op.Entry.Attributes)

	return nil
}

func (fs *helloFS) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	// Find the info for this inode
	info, ok := gInodeInfo[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	// Copy over its attributes
	op.Attributes = info.attributes

	// Patch attributes
	fs.patchAttributes(&op.Attributes)

	return nil
}

// Open a directory Inode. On Linux the sends
func (fs *helloFS) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	// Allow opening any directory
	return nil
}

func (fs *helloFS) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	// Find the info for this inode
	info, ok := gInodeInfo[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	if !info.dir {
		// Input/output error
		return fuse.EIO
	}

	// All nodes inside of the directory
	entries := info.children

	// Grab the range of interest
	if op.Offset > fuseops.DirOffset(len(entries)) {
		return fuse.EIO
	}

	entries = entries[op.Offset:]

	// Resume at the specified offset into the array
	for _, e := range entries {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], e)
		if n == 0 {
			break
		}

		op.BytesRead += n
	}

	return nil
}

func (fs *helloFS) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	// Allow opening any file
	return nil
}

func (fs *helloFS) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	// Let io.ReaderAt deal with the semantics
	reader := strings.NewReader("Hello, World!")

	var err error
	op.BytesRead, err = reader.ReadAt(op.Dst, op.Offset)

	// Special case: FUSE doens't expect us to return io.EOF.
	if err == io.EOF {
		return nil
	}

	return err
}