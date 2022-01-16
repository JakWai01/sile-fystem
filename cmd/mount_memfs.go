package cmd

import (
	"context"
	"log"

	"github.com/JakWai01/sile-fystem/internal/logging"
	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/helpers"
	"github.com/jacobsa/fuse"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	mountpoint = "mountpoint"
)

var memFsCmd = &cobra.Command{
	Use:   "memfs",
	Short: "Mount a folder on a given path",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.NewJSONLogger(5)

		serve := filesystem.NewFileSystem(helpers.CurrentUid(), helpers.CurrentGid(), viper.GetString(mountpoint), "", logger, afero.NewMemMapFs())

		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		mfs, err := fuse.Mount(viper.GetString(mountpoint), serve, cfg)
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
	memFsCmd.PersistentFlags().String(mountpoint, "", "mount")

	// Bind env variables
	if err := viper.BindPFlags(memFsCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.SetEnvPrefix("sile-fystem")
	viper.AutomaticEnv()
}
