package main

import (
	"context"
	"flag"
	"log"
	"os/user"
	"strconv"

	"github.com/JakWai01/sile-fystem/memfs"
	"github.com/jacobsa/fuse"
)

var fMountPoint = flag.String("mount_point", "", "Path to mount point.")

func main() {
	flag.Parse()

	server := memfs.NewMemFS(currentUid(), currentGid())

	// Mount the file system
	if *fMountPoint == "" {
		log.Fatalf("You must set --mount_point.")
	}

	cfg := &fuse.MountConfig{
		ReadOnly: false,
	}

	// Mount the fuse.Server we created earlier
	mfs, err := fuse.Mount(*fMountPoint, server, cfg)
	if err != nil {
		log.Fatalf("Mount: %v", err)
	}

	// Wait for it being unmounted
	if err := mfs.Join(context.Background()); err != nil {
		log.Fatalf("Join %v", err)
	}
}

func currentUid() uint32 {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	uid, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(uid)
}

func currentGid() uint32 {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	gid, err := strconv.ParseUint(user.Gid, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(gid)
}
