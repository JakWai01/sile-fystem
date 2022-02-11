package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	verboseFlag  = "verbose"
	metadataFlag = "metadata"
	mountpoint   = "mountpoint"
)

var rootCmd = &cobra.Command{
	Use:   "sile-fystem",
	Short: "sile-fystem, a file-system",
	Long: `sile-fystem, a file-system using FUSE."

For more information, please visit https://github.com/JakWai01/sile-fystem`,
}

func Execute() error {

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(home, ".local", "share", "stfs", "var", "lib", "stfs", "metadata.sqlite")

	rootCmd.PersistentFlags().IntP(verboseFlag, "v", 2, fmt.Sprintf("Verbosity level (default %v)", 2))
	rootCmd.PersistentFlags().StringP(metadataFlag, "m", metadataPath, "Metadata database to use")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	mountPath := filepath.Join(homeDir, filepath.Join("Documents", "mount"))

	os.MkdirAll(mountPath, os.ModePerm)

	rootCmd.PersistentFlags().String(mountpoint, mountPath, "Mountpoint")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		return err
	}

	viper.AutomaticEnv()

	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(mountCmd)
	rootCmd.AddCommand(memFsCmd)
	rootCmd.AddCommand(osFsCmd)
}
