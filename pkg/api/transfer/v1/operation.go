package api

import (
	"context"

	"github.com/jacobsa/fuse/fuseops"
)

type Transfer struct {
	Message
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type Exec struct {
	Message
	Context              context.Context               `json:"context"`
	LookUpInodeOp        *fuseops.LookUpInodeOp        `json:"lookupinodeop"`
	GetInodeAttributesOp *fuseops.GetInodeAttributesOp `json:"getinodeattributesop"`
	SetInodeAttributesOp *fuseops.SetInodeAttributesOp `json:"setinodeattributesop"`
	MkDirOp              *fuseops.MkDirOp              `json:"mkdirop"`
	MkNodeOp             *fuseops.MkNodeOp             `json:"mknodeop"`
	CreatFileOp          *fuseops.CreateFileOp         `json:"createfileop"`
	CreateSymlinkOp      *fuseops.CreateSymlinkOp      `json:"createsymlinkop"`
	CreateLinkOp         *fuseops.CreateLinkOp         `json:"createlinkop"`
	RenameOp             *fuseops.RenameOp             `json:"renameop"`
	RmDirOp              *fuseops.RenameOp             `json:"rmdirop"`
	UnlinkOp             *fuseops.UnlinkOp             `json:"unlinkop"`
	OpenDirOp            *fuseops.OpenDirOp            `json:"opendirop"`
	ReadDirOp            *fuseops.ReadDirOp            `json:"readdirop"`
	OpenFileOp           *fuseops.OpenFileOp           `json:"openfileop"`
	ReadFileOP           *fuseops.ReadFileOp           `json:"readfileop"`
	WriteFileOp          *fuseops.WriteFileOp          `json:"writefileop"`
	FlushFileOp          *fuseops.FlushFileOp          `json:"flushfileop"`
	ReadSymLinkOp        *fuseops.ReadSymlinkOp        `json:"readsymlinkop"`
	GetXattrOp           *fuseops.GetXattrOp           `json:"getxattrop"`
	ListXattrOp          *fuseops.ListXattrOp          `json:"listxattrop"`
	RemoveXattrOp        *fuseops.RemoveXattrOp        `json:"removexattr"`
	SetXattrOp           *fuseops.SetXattrOp           `json:"setxattrop"`
	FallocateOp          *fuseops.FallocateOp          `json:"fallocateop"`
}
