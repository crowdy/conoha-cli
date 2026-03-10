package dns

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/output"
)

var Cmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage DNS domains and records",
}

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage DNS domains",
}

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Manage DNS records",
}

func init() {
	Cmd.AddCommand(domainCmd)
	Cmd.AddCommand(recordCmd)

	// Domain commands
	domainListCmd := &cobra.Command{
		Use: "list", Short: "List domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			domains, err := api.NewDNSAPI(client).ListDomains()
			if err != nil {
				return err
			}

			type row struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				TTL  int    `json:"ttl"`
			}
			rows := make([]row, len(domains))
			for i, d := range domains {
				rows[i] = row{ID: d.ID, Name: d.Name, TTL: d.TTL}
			}
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
		},
	}

	domainShowCmd := &cobra.Command{
		Use: "show <id>", Short: "Show domain details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			domain, err := api.NewDNSAPI(client).GetDomain(args[0])
			if err != nil {
				return err
			}
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, domain)
		},
	}

	domainCreateCmd := &cobra.Command{
		Use: "create", Short: "Create a domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			email, _ := cmd.Flags().GetString("email")
			ttl, _ := cmd.Flags().GetInt("ttl")
			domain, err := api.NewDNSAPI(client).CreateDomain(name, email, ttl)
			if err != nil {
				return err
			}
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, domain)
		},
	}
	domainCreateCmd.Flags().String("name", "", "domain name (required)")
	domainCreateCmd.Flags().String("email", "", "admin email (required)")
	domainCreateCmd.Flags().Int("ttl", 3600, "TTL")
	_ = domainCreateCmd.MarkFlagRequired("name")
	_ = domainCreateCmd.MarkFlagRequired("email")

	domainDeleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a domain", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewDNSAPI(client).DeleteDomain(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Domain %s deleted\n", args[0])
			return nil
		},
	}

	domainCmd.AddCommand(domainListCmd, domainShowCmd, domainCreateCmd, domainDeleteCmd)

	// Record commands
	recordListCmd := &cobra.Command{
		Use: "list", Short: "List records",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			domainID, _ := cmd.Flags().GetString("domain-id")
			records, err := api.NewDNSAPI(client).ListRecords(domainID)
			if err != nil {
				return err
			}

			type row struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
				Data string `json:"data"`
				TTL  int    `json:"ttl"`
			}
			rows := make([]row, len(records))
			for i, r := range records {
				rows[i] = row{ID: r.ID, Name: r.Name, Type: r.Type, Data: r.Data, TTL: r.TTL}
			}
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
		},
	}
	recordListCmd.Flags().String("domain-id", "", "domain ID (required)")
	_ = recordListCmd.MarkFlagRequired("domain-id")

	recordCreateCmd := &cobra.Command{
		Use: "create", Short: "Create a record",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			domainID, _ := cmd.Flags().GetString("domain-id")
			name, _ := cmd.Flags().GetString("name")
			rType, _ := cmd.Flags().GetString("type")
			data, _ := cmd.Flags().GetString("data")
			ttl, _ := cmd.Flags().GetInt("ttl")

			var priority *int
			if cmd.Flags().Changed("priority") {
				p, _ := cmd.Flags().GetInt("priority")
				priority = &p
			}

			record, err := api.NewDNSAPI(client).CreateRecord(domainID, name, rType, data, ttl, priority)
			if err != nil {
				return err
			}
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, record)
		},
	}
	recordCreateCmd.Flags().String("domain-id", "", "domain ID (required)")
	recordCreateCmd.Flags().String("name", "", "record name (required)")
	recordCreateCmd.Flags().String("type", "", "record type (required)")
	recordCreateCmd.Flags().String("data", "", "record data (required)")
	recordCreateCmd.Flags().Int("ttl", 3600, "TTL")
	recordCreateCmd.Flags().Int("priority", 0, "priority (for MX records)")
	_ = recordCreateCmd.MarkFlagRequired("domain-id")
	_ = recordCreateCmd.MarkFlagRequired("name")
	_ = recordCreateCmd.MarkFlagRequired("type")
	_ = recordCreateCmd.MarkFlagRequired("data")

	recordDeleteCmd := &cobra.Command{
		Use: "delete <record-id>", Short: "Delete a record", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			domainID, _ := cmd.Flags().GetString("domain-id")
			if err := api.NewDNSAPI(client).DeleteRecord(domainID, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Record %s deleted\n", args[0])
			return nil
		},
	}
	recordDeleteCmd.Flags().String("domain-id", "", "domain ID (required)")
	_ = recordDeleteCmd.MarkFlagRequired("domain-id")

	recordCmd.AddCommand(recordListCmd, recordCreateCmd, recordDeleteCmd)
}
