package cmd

import (
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Image utilities",
}

func init() {
	rootCmd.AddCommand(imageCmd)
}
