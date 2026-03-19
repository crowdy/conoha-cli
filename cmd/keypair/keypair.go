package keypair

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/output"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

// uniquePath returns a path that doesn't exist yet.
// If basePath exists, it tries basePath_1, basePath_2, etc.
func uniquePath(basePath string) string {
	if _, err := os.Stat(basePath); err != nil {
		return basePath
	}
	for i := 1; ; i++ {
		p := fmt.Sprintf("%s_%d", basePath, i)
		if _, err := os.Stat(p); err != nil {
			return p
		}
	}
}

var Cmd = &cobra.Command{
	Use:   "keypair",
	Short: "Manage SSH keypairs",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)

	createCmd.Flags().String("public-key", "", "public key content")
	createCmd.Flags().StringP("output", "o", "", "save private key to file (default: ~/.ssh/conoha_<name>)")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List keypairs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)
		keypairs, err := compute.ListKeypairs()
		if err != nil {
			return err
		}

		type row struct {
			Name        string `json:"name"`
			Fingerprint string `json:"fingerprint"`
		}
		rows := make([]row, len(keypairs))
		for i, k := range keypairs {
			rows[i] = row{Name: k.Name, Fingerprint: k.Fingerprint}
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
	},
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a keypair",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)

		req := &model.KeypairCreateRequest{}
		req.Keypair.Name = args[0]
		if pk, _ := cmd.Flags().GetString("public-key"); pk != "" {
			req.Keypair.PublicKey = pk
		}

		kp, err := compute.CreateKeypair(req)
		if err != nil {
			return err
		}

		// Save keys to files if returned by API
		if kp.PrivateKey != "" {
			basePath, _ := cmd.Flags().GetString("output")
			if basePath == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("cannot determine home directory: %w", err)
				}
				basePath = filepath.Join(home, ".ssh", fmt.Sprintf("conoha_%s", args[0]))
			}
			basePath = uniquePath(basePath)
			pubPath := basePath + ".pub"

			dir := filepath.Dir(basePath)
			if err := os.MkdirAll(dir, 0700); err != nil {
				return fmt.Errorf("creating directory %s: %w", dir, err)
			}
			if err := os.WriteFile(basePath, []byte(kp.PrivateKey), 0600); err != nil {
				return fmt.Errorf("saving private key: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Private key saved to %s\n", basePath)
			if kp.PublicKey != "" {
				if err := os.WriteFile(pubPath, []byte(kp.PublicKey), 0644); err != nil {
					return fmt.Errorf("saving public key: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Public key saved to %s\n", pubPath)
			}
		}

		fmt.Fprintf(os.Stderr, "Keypair %s created\n", kp.Name)
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a keypair",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete keypair %q?", args[0]))
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
		compute := api.NewComputeAPI(client)
		if err := compute.DeleteKeypair(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Keypair %s deleted\n", args[0])
		return nil
	},
}
