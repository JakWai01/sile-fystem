package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	mountPoint = "mountpoint"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a folder on a given path",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(viper.GetString(mountPoint))
		return nil
	},
}

func init() {
	mountCmd.PersistentFlags().String(mountPoint, "", "Mountpoint")

	// Bind env variables
	if err := viper.BindPFlags(mountCmd.PersistentFlags()); err != nil {
		log.Fatal("could not bind flags:", err)
	}
	viper.SetEnvPrefix("airdrip")
	viper.AutomaticEnv()
}
