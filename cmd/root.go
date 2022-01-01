package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	verboseFlag = "verbose"
)

var rootCmd = &cobra.Command{
	Use:   "sf",
	Short: "sile-fystem, a file-system",
	Long: `sile-fystem, a file-system using FUSE."

For more information, please visit https://github.com/JakWai01/sile-fystem`,
}

func Execute() error {
	rootCmd.PersistentFlags().IntP(verboseFlag, "v", 2, fmt.Sprintf("Verbosity level (default %v)", 2, []int{0, 1, 2, 3, 4}))

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		return err
	}

	viper.AutomaticEnv()

	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(mountCmd)
}
