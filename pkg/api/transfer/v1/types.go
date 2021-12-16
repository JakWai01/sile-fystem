package api

const (
	OpcodeTransfer = "transfer"

	FuncLookUpInode        = "lookupinode"
	FuncGetInodeAttributes = "getinodeattributes"
	FuncSetInodeAttributes = "setinodeattributes"
	FuncMkDir              = "mkdir"
	FuncMkNode             = "mknode"
	FuncCreateFile         = "createfile"
	FuncCreateSymlink      = "createsymlink"
	FuncCreateLink         = "createlink"
	FuncRename             = "rename"
	FuncRmDir              = "rmdir"
	FuncUnlink             = "unlink"
	FuncOpenDir            = "opendir"
	FuncReadDir            = "readdir"
	FuncOpenFile           = "openfile"
	FuncReadFile           = "readfile"
	FuncWriteFile          = "writefile"
	FuncFlushFile          = "flushfile"
	FuncReadSymlink        = "readsymlink"
	FuncGetXattr           = "getxattr"
	FuncListXattr          = "listxattr"
	FuncRemoveXattr        = "removexattr"
	FuncSetXattr           = "setxattr"
	FuncFallocate          = "fallocate"
)
