package api

import "github.com/jacobsa/fuse/fuseops"

type GetInodeAttributesRequest struct {
	Message
	InodeID fuseops.InodeID
}

type LookUpInodeRequest struct {
	Message
	InodeID fuseops.InodeID
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
