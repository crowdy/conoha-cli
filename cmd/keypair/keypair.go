package keypair

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/output"
)

var Cmd = &cobra.Command{
	Use:   "keypair",
	Short: "Manage SSH keypairs",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)

	createCmd.Flags().String("public-key", "", "public key content")
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
	Args:  cobra.ExactArgs(1),
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
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, kp)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a keypair",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
