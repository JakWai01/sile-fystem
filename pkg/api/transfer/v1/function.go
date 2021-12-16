package api

import (
	"context"

	"github.com/jacobsa/fuse/fuseops"
)

type LookUpInode struct {
	Message
	Context       context.Context
	LookUpInodeOp *fuseops.LookUpInodeOp
}

type GetInodeAttributes struct {
	Message
	Context              context.Context
	GetInodeAttributesOp *fuseops.GetInodeAttributesOp
}

type SetInodeAttributes struct {
	Message
	Context              context.Context
	SetInodeAttributesOp *fuseops.SetInodeAttributesOp
}

type MkDir struct {
	Message
	Context context.Context
	MkDirOp *fuseops.MkDirOp
}

type MkNode struct {
	Message
	Context  context.Context
	MkNodeOp *fuseops.MkNodeOp
}

type CreateFile struct {
	Message
	Context      context.Context
	CreateFileOp *fuseops.CreateFileOp
}

type CreateSymlink struct {
	Message
	Context         context.Context
	CreateSymlinkOp *fuseops.CreateSymlinkOp
}

type CreateLink struct {
	Message
	Context      context.Context
	CreateLinkOp *fuseops.CreateLinkOp
}

type Rename struct {
	Message
	Context  context.Context
	RenameOp *fuseops.RenameOp
}

type RmDir struct {
	Message
	Context context.Context
	RmDirOp *fuseops.RmDirOp
}

type Unlink struct {
	Message
	Context  context.Context
	UnlinkOp *fuseops.UnlinkOp
}

type OpenDir struct {
	Message
	Context   context.Context
	OpenDirOp *fuseops.OpenDirOp
}

type ReadDir struct {
	Message
	Context   context.Context
	ReadDirOp *fuseops.ReadDirOp
}

type OpenFile struct {
	Message
	Context    context.Context
	OpenFileOp *fuseops.OpenFileOp
}

type ReadFile struct {
	Message
	Context    context.Context
	ReadFileOp *fuseops.ReadFileOp
}

type WriteFile struct {
	Message
	Context     context.Context
	WriteFileOp *fuseops.WriteFileOp
}

type FlushFile struct {
	Message
	Context     context.Context
	FlushFileOp *fuseops.FlushFileOp
}

type ReadSymlink struct {
	Message
	Context       context.Context
	ReadSymlinkOp *fuseops.ReadSymlinkOp
}

type GetXattr struct {
	Message
	Context    context.Context
	GetXattrOp *fuseops.GetXattrOp
}

type ListXattr struct {
	Message
	Context     context.Context
	ListXattrOp *fuseops.ListXattrOp
}

type RemoveXattr struct {
	Message
	Context       context.Context
	RemoveXattrOp *fuseops.RemoveXattrOp
}

type SetXattr struct {
	Message
	Context    context.Context
	SetXattrOp *fuseops.SetXattrOp
}

type Fallocate struct {
	Message
	Context     context.Context
	FallocateOp *fuseops.FallocateOp
}

func NewLookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) *LookUpInode {
	return &LookUpInode{Message: Message{FuncLookUpInode}, Context: ctx, LookUpInodeOp: op}
}

func NewGetInodeAttribtues(ctx context.Context, op *fuseops.GetInodeAttributesOp) *GetInodeAttributes {
	return &GetInodeAttributes{Message: Message{FuncGetInodeAttributes}, Context: ctx, GetInodeAttributesOp: op}
}

func NewSetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) *SetInodeAttributes {
	return &SetInodeAttributes{Message: Message{FuncSetInodeAttributes}, Context: ctx, SetInodeAttributesOp: op}
}

func NewMkDir(ctx context.Context, op *fuseops.MkDirOp) *MkDir {
	return &MkDir{Message: Message{FuncMkDir}, Context: ctx, MkDirOp: op}
}

func NewMkNode(ctx context.Context, op *fuseops.MkNodeOp) *MkNode {
	return &MkNode{Message: Message{FuncMkNode}, Context: ctx, MkNodeOp: op}
}

func NewCreateFile(ctx context.Context, op *fuseops.CreateFileOp) *CreateFile {
	return &CreateFile{Message: Message{FuncCreateFile}, Context: ctx, CreateFileOp: op}
}

func NewCreateSymLink(ctx context.Context, op *fuseops.CreateSymlinkOp) *CreateSymlink {
	return &CreateSymlink{Message: Message{FuncCreateSymlink}, Context: ctx, CreateSymlinkOp: op}
}

func NewCreateLink(ctx context.Context, op *fuseops.CreateLinkOp) *CreateLink {
	return &CreateLink{Message: Message{FuncCreateLink}, Context: ctx, CreateLinkOp: op}
}

func NewRename(ctx context.Context, op *fuseops.RenameOp) *Rename {
	return &Rename{Message: Message{FuncRename}, Context: ctx, RenameOp: op}
}

func NewRmDir(ctx context.Context, op *fuseops.RmDirOp) *RmDir {
	return &RmDir{Message: Message{FuncRmDir}, Context: ctx, RmDirOp: op}
}

func NewUnlink(ctx context.Context, op *fuseops.UnlinkOp) *Unlink {
	return &Unlink{Message: Message{FuncUnlink}, Context: ctx, UnlinkOp: op}
}

func NewOpenDir(ctx context.Context, op *fuseops.OpenDirOp) *OpenDir {
	return &OpenDir{Message: Message{FuncOpenDir}, Context: ctx, OpenDirOp: op}
}

func NewReadDir(ctx context.Context, op *fuseops.ReadDirOp) *ReadDir {
	return &ReadDir{Message: Message{FuncReadDir}, Context: ctx, ReadDirOp: op}
}

func NewOpenFile(ctx context.Context, op *fuseops.OpenFileOp) *OpenFile {
	return &OpenFile{Message: Message{FuncOpenFile}, Context: ctx, OpenFileOp: op}
}

func NewReadFile(ctx context.Context, op *fuseops.ReadFileOp) *ReadFile {
	return &ReadFile{Message: Message{FuncReadFile}, Context: ctx, ReadFileOp: op}
}

func NewWriteFile(ctx context.Context, op *fuseops.WriteFileOp) *WriteFile {
	return &WriteFile{Message: Message{FuncWriteFile}, Context: ctx, WriteFileOp: op}
}

func NewFlushFile(ctx context.Context, op *fuseops.FlushFileOp) *FlushFile {
	return &FlushFile{Message: Message{FuncFlushFile}, Context: ctx, FlushFileOp: op}
}

func NewReadSymlink(ctx context.Context, op *fuseops.ReadSymlinkOp) *ReadSymlink {
	return &ReadSymlink{Message: Message{FuncReadSymlink}, Context: ctx, ReadSymlinkOp: op}
}

func NewGetXattr(ctx context.Context, op *fuseops.GetXattrOp) *GetXattr {
	return &GetXattr{Message: Message{FuncGetXattr}, Context: ctx, GetXattrOp: op}
}

func NewRemoveXattr(ctx context.Context, op *fuseops.RemoveXattrOp) *RemoveXattr {
	return &RemoveXattr{Message: Message{FuncRemoveXattr}, Context: ctx, RemoveXattrOp: op}
}

func NewSetXattr(ctx context.Context, op *fuseops.SetXattrOp) *SetXattr {
	return &SetXattr{Message: Message{FuncSetXattr}, Context: ctx, SetXattrOp: op}
}

func NewFallocate(ctx context.Context, op *fuseops.FallocateOp) *Fallocate {
	return &Fallocate{Message: Message{FuncFallocate}, Context: ctx, FallocateOp: op}
}
