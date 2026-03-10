package cmdutil

import (
	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/config"
	cerrors "github.com/crowdy/conoha-cli/internal/errors"
)

// NewClient creates an API client from the cobra command context.
func NewClient(cmd *cobra.Command) (*api.Client, error) {
	profileName, _ := cmd.Flags().GetString("profile")
	if profileName == "" {
		profileName = config.EnvOr(config.EnvProfile, "")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if profileName == "" {
		profileName = cfg.ActiveProfile
	}
	if profileName == "" {
		profileName = "default"
	}

	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return nil, &cerrors.ConfigError{Message: "profile not found, run 'conoha auth login'"}
	}

	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}

	tokens, err := config.LoadTokens()
	if err != nil {
		return nil, err
	}

	token, err := api.EnsureToken(profileName, cfg, creds, tokens)
	if err != nil {
		return nil, err
	}

	region := profile.Region
	if region == "" {
		region = config.DefaultRegion
	}

	return api.NewClient(region, token, profile.TenantID), nil
}

// GetFormat returns the output format from flags.
func GetFormat(cmd *cobra.Command) string {
	format, _ := cmd.Flags().GetString("format")
	if format != "" {
		return format
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
