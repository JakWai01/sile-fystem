package cmd

import (
	"context"
	"log"

	server "github.com/JakWai01/sile-fystem/pkg/client"
	"github.com/jacobsa/fuse"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve a folder on a given path",
	RunE: func(cmd *cobra.Command, args []string) error {

		serve := server.NewFileSystem(currentUid(), currentGid(), viper.GetString(Servepoint))

		cfg := &fuse.MountConfig{
			ReadOnly:                  false,
			DisableDefaultPermissions: false,
		}

		// Mount the fuse.Server we created earlier
		mfs, err := fuse.Mount(viper.GetString(Servepoint), serve, cfg)
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
	serveCmd.PersistentFlags().String(Servepoint, "", "Servepoint")

	// Bind env variables
	if err := viper.BindPFlags(serveCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.SetEnvPrefix("sile-fystem")
	viper.AutomaticEnv()
}
