package api

import "github.com/jacobsa/fuse/fuseops"

type GetInodeAttributesRequest struct {
	Message
	InodeID fuseops.InodeID
}

type LookUpInodeRequest struct {
	Message
	InodeID fuseops.InodeID
	Name    string
}

type OpenDirRequest struct {
	Message
	InodeID fuseops.InodeID
}

type ReadDirRequest struct {
	Message
	InodeID fuseops.InodeID
}

type MkDirRequest struct {
	Message
	InodeID fuseops.InodeID
	Name    string
}

type GetInodeAttributesResponse struct {
	Message
	Inode Inode
}

type LookUpInodeResponse struct {
	Message
}

type OpenDirResponse struct {
	Message
}

type ReadDirResponse struct {
	Message
}

type MkDirResponse struct {
	Message
}

func NewGetInodeAttributesRequest(inodeID fuseops.InodeID) *GetInodeAttributesRequest {
	return &GetInodeAttributesRequest{Message: Message{FuncGetInodeAttributes}, InodeID: inodeID}
}

func NewLookUpInodeRequest(inodeID fuseops.InodeID, name string) *LookUpInodeRequest {
	return &LookUpInodeRequest{Message: Message{FuncLookUpInode}, InodeID: inodeID, Name: name}
}

func NewOpenDirRequest(inodeID fuseops.InodeID) *OpenDirRequest {
	return &OpenDirRequest{Message: Message{FuncOpenDir}, InodeID: inodeID}
}

func NewReadDirRequest(inodeID fuseops.InodeID) *ReadDirRequest {
	return &ReadDirRequest{Message: Message{FuncReadDir}, InodeID: inodeID}
}

func NewMkDirRequest(inodeID fuseops.InodeID, name string) *MkDirRequest {
	return &MkDirRequest{Message: Message{FuncMkDir}, InodeID: inodeID, Name: name}
}
