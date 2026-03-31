package server

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

func init() {
	addSecurityGroupCmd.Flags().String("name", "", "security group name")
	_ = addSecurityGroupCmd.MarkFlagRequired("name")

	removeSecurityGroupCmd.Flags().String("name", "", "security group name")
	_ = removeSecurityGroupCmd.MarkFlagRequired("name")
}

var addSecurityGroupCmd = &cobra.Command{
	Use:     "add-security-group <id|name>",
	Aliases: []string{"add-sg"},
	Short:   "Add a security group to a server",
	Args:    cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		ok, err := prompt.Confirm(fmt.Sprintf("Add security group %q to server %s?", name, args[0]))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if err := compute.AddSecurityGroup(id, name); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Security group %q added to server %s\n", name, args[0])
		return nil
	},
}

var removeSecurityGroupCmd = &cobra.Command{
	Use:     "remove-security-group <id|name>",
	Aliases: []string{"remove-sg"},
	Short:   "Remove a security group from a server",
	Args:    cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		ok, err := prompt.Confirm(fmt.Sprintf("Remove security group %q from server %s?", name, args[0]))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if err := compute.RemoveSecurityGroup(id, name); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Security group %q removed from server %s\n", name, args[0])
		return nil
	},
}
