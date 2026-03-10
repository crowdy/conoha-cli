package volume

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
	Use:   "volume",
	Short: "Manage block storage volumes",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(typesCmd)
	Cmd.AddCommand(backupCmd)

	createCmd.Flags().String("name", "", "volume name")
	createCmd.Flags().Int("size", 0, "volume size in GB (required)")
	createCmd.Flags().String("type", "", "volume type")
	createCmd.Flags().String("description", "", "volume description")
	_ = createCmd.MarkFlagRequired("size")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		volumes, err := api.NewVolumeAPI(client).ListVolumes()
		if err != nil {
			return err
		}

		type row struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Size   int    `json:"size"`
		}
		rows := make([]row, len(volumes))
		for i, v := range volumes {
			rows[i] = row{ID: v.ID, Name: v.Name, Status: v.Status, Size: v.Size}
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show volume details",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		vol, err := api.NewVolumeAPI(client).GetVolume(args[0])
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, vol)
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a volume",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		size, _ := cmd.Flags().GetInt("size")
		volType, _ := cmd.Flags().GetString("type")
		desc, _ := cmd.Flags().GetString("description")

		req := &model.VolumeCreateRequest{}
		req.Volume.Name = name
		req.Volume.Size = size
		req.Volume.VolumeType = volType
		req.Volume.Description = desc

		vol, err := api.NewVolumeAPI(client).CreateVolume(req)
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, vol)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a volume",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		if err := api.NewVolumeAPI(client).DeleteVolume(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Volume %s deleted\n", args[0])
		return nil
	},
}

var typesCmd = &cobra.Command{
	Use:   "types",
	Short: "List volume types",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		types, err := api.NewVolumeAPI(client).ListVolumeTypes()
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, types)
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage volume backups",
}

func init() {
	backupListCmd := &cobra.Command{
		Use:   "list",
		Short: "List backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			backups, err := api.NewVolumeAPI(client).ListBackups()
			if err != nil {
				return err
			}

			type row struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Status   string `json:"status"`
				VolumeID string `json:"volume_id"`
				Size     int    `json:"size"`
			}
			rows := make([]row, len(backups))
			for i, b := range backups {
				rows[i] = row{ID: b.ID, Name: b.Name, Status: b.Status, VolumeID: b.VolumeID, Size: b.Size}
			}
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
		},
	}

	backupShowCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show backup details",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			backup, err := api.NewVolumeAPI(client).GetBackup(args[0])
			if err != nil {
				return err
			}
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, backup)
		},
	}

	backupRestoreCmd := &cobra.Command{
		Use:   "restore <backup-id> <volume-id>",
		Short: "Restore a backup to a volume",
		Args:  cmdutil.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewVolumeAPI(client).RestoreBackup(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Backup %s restoring to volume %s\n", args[0], args[1])
			return nil
		},
	}

	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupShowCmd)
	backupCmd.AddCommand(backupRestoreCmd)
}
