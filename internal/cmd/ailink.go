package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/config"
)

var ailinkCmd = &cobra.Command{
	Use:   "ailink",
	Short: "Manage expert prompts",
}

var ailinkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available prompts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg, err := config.Load(ctx)
		if err != nil {
			return err
		}

		registry, err := buildPromptRegistry(cfg)
		if err != nil {
			return err
		}

		prompts := registry.List()
		if len(prompts) == 0 {
			fmt.Println("No prompts found.")
			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "SLUG\tVERSION\tDESCRIPTION") // nolint:errcheck // tabwriter buffers; errors surface at Flush
		for _, prompt := range prompts {
			if prompt == nil {
				continue
			}
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", prompt.Config.Slug, prompt.Config.Version, prompt.Config.Description) // nolint:errcheck // tabwriter buffers
		}
		return writer.Flush()
	},
}

func init() {
	rootCmd.AddCommand(ailinkCmd)
	ailinkCmd.AddCommand(ailinkListCmd)
}
