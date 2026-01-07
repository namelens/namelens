package cmd

import "github.com/spf13/cobra"

var rateLimitCmd = &cobra.Command{
	Use:   "rate-limit",
	Short: "Manage persisted rate limit state",
}

func init() {
	rateLimitCmd.AddCommand(rateLimitListCmd)
	rateLimitCmd.AddCommand(rateLimitResetCmd)
	rootCmd.AddCommand(rateLimitCmd)
}
