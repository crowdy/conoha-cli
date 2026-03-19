package identity

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

var Cmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage identity resources",
}

var credentialCmd = &cobra.Command{Use: "credential", Short: "Manage credentials"}
var subuserCmd = &cobra.Command{Use: "subuser", Short: "Manage sub-users"}
var roleCmd = &cobra.Command{Use: "role", Short: "Manage roles"}

func init() {
	Cmd.AddCommand(credentialCmd)
	Cmd.AddCommand(subuserCmd)
	Cmd.AddCommand(roleCmd)

	// Credential commands
	credListCmd := &cobra.Command{
		Use: "list", Short: "List credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			creds, err := api.NewIdentityAPI(client).ListCredentials()
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, creds)
		},
	}

	credShowCmd := &cobra.Command{
		Use: "show <id>", Short: "Show credential details", Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			cred, err := api.NewIdentityAPI(client).GetCredential(args[0])
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, cred)
		},
	}

	credDeleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a credential", Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete credential %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewIdentityAPI(client).DeleteCredential(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Credential %s deleted\n", args[0])
			return nil
		},
	}

	credentialCmd.AddCommand(credListCmd, credShowCmd, credDeleteCmd)

	// Sub-user commands
	subuserListCmd := &cobra.Command{
		Use: "list", Short: "List sub-users",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			users, err := api.NewIdentityAPI(client).ListSubUsers()
			if err != nil {
				return err
			}

			type row struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Enabled bool   `json:"enabled"`
			}
			rows := make([]row, len(users))
			for i, u := range users {
				rows[i] = row{ID: u.ID, Name: u.Name, Enabled: u.Enabled}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}

	subuserDeleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a sub-user", Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete sub-user %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewIdentityAPI(client).DeleteSubUser(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Sub-user %s deleted\n", args[0])
			return nil
		},
	}

	subuserCmd.AddCommand(subuserListCmd, subuserDeleteCmd)

	// Role commands
	roleListCmd := &cobra.Command{
		Use: "list", Short: "List roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			roles, err := api.NewIdentityAPI(client).ListRoles()
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, roles)
		},
	}

	roleCmd.AddCommand(roleListCmd)
}
