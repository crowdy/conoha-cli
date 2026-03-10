package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	iconfig "github.com/crowdy/conoha-cli/internal/config"
)

// Cmd is the config command group.
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

func init() {
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(setCmd)
	Cmd.AddCommand(pathCmd)
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := iconfig.Load()
		if err != nil {
			return err
		}

		fmt.Printf("Config dir:     %s\n", iconfig.DefaultConfigDir())
		fmt.Printf("Active profile: %s\n", cfg.ActiveProfile)
		fmt.Printf("Default format: %s\n", cfg.Defaults.Format)
		fmt.Printf("Profiles:       %d\n", len(cfg.Profiles))

		for name, p := range cfg.Profiles {
			marker := " "
			if name == cfg.ActiveProfile {
				marker = "*"
			}
			fmt.Printf("  %s %s (tenant=%s, region=%s)\n", marker, name, p.TenantID, p.Region)
		}
		return nil
	},
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  "Set a configuration value. Keys: format, region",
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		cfg, err := iconfig.Load()
		if err != nil {
			return err
		}

		switch key {
		case "format":
			cfg.Defaults.Format = value
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		return cfg.Save()
	},
}

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print config directory path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(iconfig.DefaultConfigDir())
	},
}
