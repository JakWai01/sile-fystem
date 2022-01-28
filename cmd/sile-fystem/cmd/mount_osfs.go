package cmd

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/JakWai01/sile-fystem/internal/logging"
	"github.com/JakWai01/sile-fystem/pkg/filesystem"
	"github.com/JakWai01/sile-fystem/pkg/posix"
	"github.com/jacobsa/fuse"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	storageFlag = "storage"
)

var osFsCmd = &cobra.Command{
	Use:   "osfs",
	Short: "Mount a folder on a given path using afero.OsFs as backend",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.NewJSONLogger(5)

		os.MkdirAll(viper.GetString(storageFlag), os.ModePerm)
		os.MkdirAll(viper.GetString(mountpoint), os.ModePerm)

		serve := filesystem.NewFileSystem(posix.CurrentUid(), posix.CurrentGid(), viper.GetString(mountpoint), viper.GetString(storageFlag), logger, afero.NewOsFs())

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
	osFsCmd.PersistentFlags().String(storageFlag, filepath.Join(os.TempDir(), "drive"), "Declare folder where data is stored")

	if err := viper.BindPFlags(osFsCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.SetEnvPrefix("sile-fystem")
	viper.AutomaticEnv()
}
