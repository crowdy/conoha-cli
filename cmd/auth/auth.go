package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/config"
	cerrors "github.com/crowdy/conoha-cli/internal/errors"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

// Cmd is the auth command group.
var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

func init() {
	Cmd.AddCommand(loginCmd)
	Cmd.AddCommand(logoutCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(switchCmd)
	Cmd.AddCommand(tokenCmd)
	Cmd.AddCommand(removeCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with credentials and obtain a token",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		profileName := getProfileFlag(cmd)

		// Get existing profile or create new
		profile := cfg.Profiles[profileName]

		// Prompt for missing values
		tenantID := config.EnvOr(config.EnvTenantID, profile.TenantID)
		if tenantID == "" {
			tenantID, err = prompt.String("Tenant ID")
			if err != nil {
				return err
			}
		}

		username := config.EnvOr(config.EnvUsername, profile.Username)
		if username == "" {
			username, err = prompt.String("API Username")
			if err != nil {
				return err
			}
		}

		password := config.EnvOr(config.EnvPassword, "")
		if password == "" {
			password, err = prompt.Password("API Password")
			if err != nil {
				return err
			}
		}

		region := profile.Region
		if region == "" {
			region = config.DefaultRegion
		}

		// Authenticate
		fmt.Fprintf(os.Stderr, "Authenticating as %s...\n", username)
		result, err := api.Authenticate(region, tenantID, username, password)
		if err != nil {
			return err
		}

		// Save profile
		if cfg.Profiles == nil {
			cfg.Profiles = map[string]config.Profile{}
		}
		cfg.Profiles[profileName] = config.Profile{
			TenantID: tenantID,
			Username: username,
			Region:   region,
		}
		if cfg.ActiveProfile == "" {
			cfg.ActiveProfile = profileName
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		// Save credentials
		creds, err := config.LoadCredentials()
		if err != nil {
			return err
		}
		creds.Set(profileName, config.Credentials{Password: password})
		if err := creds.Save(); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}

		// Save token
		tokens, err := config.LoadTokens()
		if err != nil {
			return err
		}
		tokens.Set(profileName, config.TokenEntry{
			Token:     result.Token,
			ExpiresAt: result.ExpiresAt,
		})
		if err := tokens.Save(); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}

		jst := time.FixedZone("JST", 9*60*60)
		fmt.Fprintf(os.Stderr, "Logged in to profile %q (token expires %s / %s JST)\n",
			profileName,
			result.ExpiresAt.Format(time.RFC3339),
			result.ExpiresAt.In(jst).Format("2006-01-02 15:04"))
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove token and credentials for the active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := getProfileFlag(cmd)

		tokens, err := config.LoadTokens()
		if err != nil {
			return err
		}
		tokens.Delete(profileName)
		if err := tokens.Save(); err != nil {
			return err
		}

		creds, err := config.LoadCredentials()
		if err != nil {
			return err
		}
		creds.Delete(profileName)
		if err := creds.Save(); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Logged out of profile %q\n", profileName)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		profileName := getProfileFlag(cmd)
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Profile %q: not configured\n", profileName)
			return &cerrors.ConfigError{Message: fmt.Sprintf("profile %q not found", profileName)}
		}

		tokens, err := config.LoadTokens()
		if err != nil {
			return err
		}

		fmt.Printf("Profile:   %s\n", profileName)
		fmt.Printf("Tenant ID: %s\n", profile.TenantID)
		fmt.Printf("Username:  %s\n", profile.Username)
		fmt.Printf("Region:    %s\n", profile.Region)

		if entry, ok := tokens.Get(profileName); ok {
			jst := time.FixedZone("JST", 9*60*60)
			remaining := time.Until(entry.ExpiresAt)
			if remaining > 0 {
				fmt.Printf("Token:     valid (expires in %s, %s JST)\n",
					remaining.Truncate(time.Minute),
					entry.ExpiresAt.In(jst).Format("2006-01-02 15:04"))
			} else {
				fmt.Printf("Token:     expired (%s ago, was %s JST)\n",
					(-remaining).Truncate(time.Minute),
					entry.ExpiresAt.In(jst).Format("2006-01-02 15:04"))
			}
		} else {
			fmt.Printf("Token:     none\n")
		}

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		tokens, err := config.LoadTokens()
		if err != nil {
			return err
		}

		if len(cfg.Profiles) == 0 {
			fmt.Fprintln(os.Stderr, "No profiles configured. Run 'conoha auth login' to create one.")
			return nil
		}

		for name, profile := range cfg.Profiles {
			marker := " "
			if name == cfg.ActiveProfile {
				marker = "*"
			}
			tokenStatus := "no token"
			if tokens.IsValid(name) {
				tokenStatus = "authenticated"
			} else if _, ok := tokens.Get(name); ok {
				tokenStatus = "expired"
			}
			fmt.Printf("%s %s\t%s\t%s\t%s\n", marker, name, profile.Username, profile.Region, tokenStatus)
		}
		return nil
	},
}

var switchCmd = &cobra.Command{
	Use:   "switch <profile>",
	Short: "Switch active profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if _, ok := cfg.Profiles[name]; !ok {
			return &cerrors.ConfigError{Message: fmt.Sprintf("profile %q not found", name)}
		}

		cfg.ActiveProfile = name
		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Switched to profile %q\n", name)
		return nil
	},
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print current token to stdout (for scripting)",
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := getProfileFlag(cmd)

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		creds, err := config.LoadCredentials()
		if err != nil {
			return err
		}

		tokens, err := config.LoadTokens()
		if err != nil {
			return err
		}

		token, err := api.EnsureToken(profileName, cfg, creds, tokens)
		if err != nil {
			return err
		}

		fmt.Print(token)
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <profile>",
	Short: "Completely remove a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		delete(cfg.Profiles, name)
		if cfg.ActiveProfile == name {
			cfg.ActiveProfile = ""
			// Set first remaining profile as active
			for k := range cfg.Profiles {
				cfg.ActiveProfile = k
				break
			}
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		creds, err := config.LoadCredentials()
		if err != nil {
			return err
		}
		creds.Delete(name)
		_ = creds.Save()

		tokens, err := config.LoadTokens()
		if err != nil {
			return err
		}
		tokens.Delete(name)
		_ = tokens.Save()

		fmt.Fprintf(os.Stderr, "Removed profile %q\n", name)
		return nil
	},
}

func getProfileFlag(cmd *cobra.Command) string {
	if p, _ := cmd.Flags().GetString("profile"); p != "" {
		return p
	}
	if p := config.EnvOr(config.EnvProfile, ""); p != "" {
		return p
	}
	cfg, _ := config.Load()
	if cfg != nil && cfg.ActiveProfile != "" {
		return cfg.ActiveProfile
	}
	return "default"
}
