package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sf",
	Short: "sile-fystem, a file-system",
	Long: `sile-fystem, a file-system using FUSE."

For more information, please visit https://github.com/JakWai01/sile-fystem`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.AddCommand(mountCmd)
	rootCmd.AddCommand(serveCmd)
}
