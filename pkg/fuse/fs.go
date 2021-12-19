package client

import (
	"context"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

type InodeNotFound struct{}

func (e *InodeNotFound) Error() string {
	return "Inode not found in FUSE"
}

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	uid uint32
	gid uint32
}

func NewFileSystem(uid uint32, gid uint32, name string) fuse.Server {
	fs := &fileSystem{
		uid: uid,
		gid: gid,
	}

	return fuseutil.NewFileSystemServer(fs)
}

func (fs *fileSystem) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	return nil
}

func (fs *fileSystem) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	return nil
}

func (fs *fileSystem) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) error {
	return nil
}

func (fs *fileSystem) MkDir(ctx context.Context, op *fuseops.MkDirOp) error {
	return nil
}

func (fs *fileSystem) MkNode(ctx context.Context, op *fuseops.MkNodeOp) error {
	return nil
}

func (fs *fileSystem) CreateFile(ctx context.Context, op *fuseops.CreateFileOp) (err error) {
	return nil
}

func (fs *fileSystem) CreateSymlink(ctx context.Context, op *fuseops.CreateSymlinkOp) error {
	return nil
}

func (fs *fileSystem) CreateLink(ctx context.Context, op *fuseops.CreateLinkOp) error {
	return nil
}

func (fs *fileSystem) Rename(ctx context.Context, op *fuseops.RenameOp) error {
	return nil
}

func (fs *fileSystem) RmDir(ctx context.Context, op *fuseops.RmDirOp) error {
	return nil
}

func (fs *fileSystem) Unlink(ctx context.Context, op *fuseops.UnlinkOp) error {
	return nil
}

func (fs *fileSystem) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return nil
}

func (fs *fileSystem) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	return nil
}

func (fs *fileSystem) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return nil
}

func (fs *fileSystem) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	return nil
}

func (fs *fileSystem) WriteFile(ctx context.Context, op *fuseops.WriteFileOp) error {
	return nil
}

func (fs *fileSystem) FlushFile(ctx context.Context, op *fuseops.FlushFileOp) (err error) {
	return nil
}

func (fs *fileSystem) ReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) error {
	return nil
}

func (fs *fileSystem) GetXattr(ctx context.Context, op *fuseops.GetXattrOp) error {
	return nil
}

func (fs *fileSystem) ListXattr(ctx context.Context, op *fuseops.ListXattrOp) error {
	return nil
}

func (fs *fileSystem) RemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) error {
	return nil
}

func (fs *fileSystem) SetXattr(ctx context.Context, op *fuseops.SetXattrOp) error {
	return nil
}

func (fs *fileSystem) Fallocate(ctx context.Context, op *fuseops.FallocateOp) error {
	return nil
}
