package main

import (
	"context"
	"flag"
	"log"

	"github.com/JakWai01/sile-fystem/memfs"
	"github.com/jacobsa/fuse"
)

var fMountPoint = flag.String("mount_point", "", "Path to mount point.")

func main() {
	flag.Parse()

	server := memfs.NewMemFS(1, 1)

	// Mount the file system
	if *fMountPoint == "" {
		log.Fatalf("You must set --mount_point.")
	}

	cfg := &fuse.MountConfig{}

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
