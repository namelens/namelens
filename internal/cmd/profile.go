package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/namelens/namelens/internal/core"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage check profiles",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		store, err := openStore(ctx)
		if err != nil {
			return err
		}
		defer store.Close() // nolint:errcheck // best-effort cleanup; errors logged internally

		profiles, err := store.ListProfiles(ctx)
		if err != nil {
			return err
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found.")
			return nil
		}

		fmt.Println("Profiles:")
		for _, record := range profiles {
			suffix := ""
			if record.IsBuiltin {
				suffix = " (builtin)"
			}
			fmt.Printf("- %s%s\n", record.Profile.Name, suffix)
		}
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if name == "" {
			return errors.New("profile name is required")
		}

		ctx := cmd.Context()
		store, err := openStore(ctx)
		if err != nil {
			return err
		}
		defer store.Close() // nolint:errcheck // best-effort cleanup; errors logged internally

		record, err := store.GetProfile(ctx, name)
		if err != nil {
			return err
		}
		if record == nil {
			return fmt.Errorf("profile %q not found", name)
		}

		printProfile(record.Profile, record.IsBuiltin)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
}

func printProfile(profile core.Profile, builtin bool) {
	fmt.Printf("Profile: %s\n", profile.Name)
	if builtin {
		fmt.Println("Type: builtin")
	}
	if profile.Description != "" {
		fmt.Printf("Description: %s\n", profile.Description)
	}
	if len(profile.TLDs) > 0 {
		fmt.Printf("Domains: %s\n", strings.Join(profile.TLDs, ", "))
	}
	if len(profile.Registries) > 0 {
		fmt.Printf("Registries: %s\n", strings.Join(profile.Registries, ", "))
	}
	if len(profile.Handles) > 0 {
		fmt.Printf("Handles: %s\n", strings.Join(profile.Handles, ", "))
	}
}
