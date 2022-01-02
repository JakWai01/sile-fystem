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
)

var rootCmd = &cobra.Command{
	Use:   "sf",
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

	rootCmd.PersistentFlags().IntP(verboseFlag, "v", 2, fmt.Sprintf("Verbosity level (default %v)", 2, []int{0, 1, 2, 3, 4}))
	rootCmd.PersistentFlags().StringP(metadataFlag, "m", metadataPath, "Metadata database to use")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		return err
	}

	viper.AutomaticEnv()

	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(mountCmd)
}
