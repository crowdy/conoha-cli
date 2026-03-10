package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/auth"
	cmdconfig "github.com/crowdy/conoha-cli/cmd/config"
	"github.com/crowdy/conoha-cli/cmd/dns"
	"github.com/crowdy/conoha-cli/cmd/flavor"
	"github.com/crowdy/conoha-cli/cmd/identity"
	"github.com/crowdy/conoha-cli/cmd/image"
	"github.com/crowdy/conoha-cli/cmd/keypair"
	"github.com/crowdy/conoha-cli/cmd/lb"
	"github.com/crowdy/conoha-cli/cmd/network"
	"github.com/crowdy/conoha-cli/cmd/server"
	"github.com/crowdy/conoha-cli/cmd/storage"
	"github.com/crowdy/conoha-cli/cmd/volume"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/config"
	cerrors "github.com/crowdy/conoha-cli/internal/errors"
)

var (
	version = "dev"

	flagProfile string
	flagFormat  string
	flagNoInput bool
	flagQuiet   bool
	flagVerbose bool
	flagNoColor bool
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:           "conoha",
	Short:         "ConoHa VPS3 CLI",
	Long:          "Command-line interface for ConoHa VPS3 API",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		api.UserAgent = "crowdy/conoha-cli/" + version
		if flagVerbose {
			api.SetDebugLevel(api.DebugVerbose)
		}
		if flagNoInput {
			_ = os.Setenv(config.EnvNoInput, "1")
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagProfile, "profile", "", "config profile to use")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "", "output format: table, json, yaml, csv")
	rootCmd.PersistentFlags().BoolVar(&flagNoInput, "no-input", false, "disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable color output")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(auth.Cmd)
	rootCmd.AddCommand(cmdconfig.Cmd)
	rootCmd.AddCommand(server.Cmd)
	rootCmd.AddCommand(flavor.Cmd)
	rootCmd.AddCommand(keypair.Cmd)
	rootCmd.AddCommand(volume.Cmd)
	rootCmd.AddCommand(image.Cmd)
	rootCmd.AddCommand(network.Cmd)
	rootCmd.AddCommand(lb.Cmd)
	rootCmd.AddCommand(dns.Cmd)
	rootCmd.AddCommand(storage.Cmd)
	rootCmd.AddCommand(identity.Cmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cerrors.GetExitCode(err))
	}
}

// GetProfile returns the active profile name.
func GetProfile() string {
	if flagProfile != "" {
		return flagProfile
	}
	if p := config.EnvOr(config.EnvProfile, ""); p != "" {
		return p
	}
	cfg, err := config.Load()
	if err != nil {
		return "default"
	}
	if cfg.ActiveProfile != "" {
		return cfg.ActiveProfile
	}
	return "default"
}

// GetFormat returns the output format.
func GetFormat() string {
	if flagFormat != "" {
		return flagFormat
	}
	if f := config.EnvOr(config.EnvFormat, ""); f != "" {
		return f
	}
	cfg, err := config.Load()
	if err != nil {
		return config.DefaultFormat
	}
	return cfg.Defaults.Format
}

// IsQuiet returns whether quiet mode is enabled.
func IsQuiet() bool {
	return flagQuiet
}

// IsVerbose returns whether verbose mode is enabled.
func IsVerbose() bool {
	return flagVerbose
}
