package cmd

import (
	"context"
	"log"
	"os/user"
	"strconv"

	client "github.com/JakWai01/sile-fystem/pkg/server"
	"github.com/jacobsa/fuse"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a folder on a given path",
	RunE: func(cmd *cobra.Command, args []string) error {

		serve := client.NewFileSystem(currentUid(), currentGid(), viper.GetString(Mountpoint))

		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		// Mount the fuse.Server we created earlier
		mfs, err := fuse.Mount(viper.GetString(Mountpoint), serve, cfg)
		if err != nil {
			log.Fatalf("Mount: %v", err)
		}

		if err := mfs.Join(context.Background()); err != nil {
			log.Fatalf("Join %v", err)
		}

		return nil
	},
}

func init() {
	mountCmd.PersistentFlags().String(Mountpoint, "", "Mountpoint")

	// Bind env variables
	if err := viper.BindPFlags(mountCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.SetEnvPrefix("sile-fystem")
	viper.AutomaticEnv()
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
