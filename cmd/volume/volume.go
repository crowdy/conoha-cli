package volume

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

// findVolume resolves a volume by UUID or name.
func findVolume(volumeAPI *api.VolumeAPI, idOrName string) (*model.Volume, error) {
	volumes, err := volumeAPI.ListVolumes()
	if err != nil {
		return nil, err
	}
	// Try exact ID match
	for i := range volumes {
		if volumes[i].ID == idOrName {
			return &volumes[i], nil
		}
	}
	// Try name match
	var matched []*model.Volume
	for i := range volumes {
		if volumes[i].Name == idOrName {
			matched = append(matched, &volumes[i])
		}
	}
	if len(matched) == 1 {
		return matched[0], nil
	}
	if len(matched) > 1 {
		ids := make([]string, len(matched))
		for i, v := range matched {
			ids[i] = v.ID
		}
		return nil, fmt.Errorf("multiple volumes found with name %q (%s), use UUID instead", idOrName, strings.Join(ids, ", "))
	}
	return nil, fmt.Errorf("volume %q not found", idOrName)
}

var Cmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage block storage volumes",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(renameCmd)
	Cmd.AddCommand(typesCmd)
	Cmd.AddCommand(backupCmd)

	createCmd.Flags().String("name", "", "volume name (required)")
	createCmd.Flags().Int("size", 0, "volume size in GB (required)")
	createCmd.Flags().String("type", "", "volume type")
	createCmd.Flags().String("description", "", "volume description")
	_ = createCmd.MarkFlagRequired("name")
	_ = createCmd.MarkFlagRequired("size")
	cmdutil.AddWaitFlags(createCmd)

	renameCmd.Flags().String("name", "", "new volume name")
	renameCmd.Flags().String("description", "", "new volume description")
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
		return cmdutil.FormatOutput(cmd, rows)
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
		return cmdutil.FormatOutput(cmd, vol)
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

		volumeAPI := api.NewVolumeAPI(client)
		vol, err := volumeAPI.CreateVolume(req)
		if err != nil {
			return err
		}

		if wc := cmdutil.GetWaitConfig(cmd, "volume "+name); wc != nil {
			fmt.Fprintf(os.Stderr, "Waiting for volume %s to become available...\n", name)
			if err := waitForVolumeAvailable(volumeAPI, vol.ID, wc); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Volume %s is available.\n", name)
		}

		return cmdutil.FormatOutput(cmd, vol)
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
		volumeAPI := api.NewVolumeAPI(client)
		vol, err := volumeAPI.GetVolume(args[0])
		if err != nil {
			return err
		}
		label := fmt.Sprintf("Delete volume %q (%s)?", vol.Name, vol.ID)
		if vol.Name == "" {
			label = fmt.Sprintf("Delete volume %s?", vol.ID)
		}
		ok, err := prompt.Confirm(label)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if err := volumeAPI.DeleteVolume(vol.ID); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Volume %s deleted\n", args[0])
		return nil
	},
}

var renameCmd = &cobra.Command{
	Use:   "rename <id|name>",
	Short: "Rename a volume",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newName, _ := cmd.Flags().GetString("name")
		newDesc, _ := cmd.Flags().GetString("description")

		if newName == "" && newDesc == "" {
			return fmt.Errorf("at least one of --name or --description is required")
		}

		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		volumeAPI := api.NewVolumeAPI(client)
		vol, err := findVolume(volumeAPI, args[0])
		if err != nil {
			return err
		}

		body := map[string]any{}
		if newName != "" {
			body["name"] = newName
		}
		if newDesc != "" {
			body["description"] = newDesc
		}
		if err := volumeAPI.UpdateVolume(vol.ID, body); err != nil {
			return err
		}

		updated, err := volumeAPI.GetVolume(vol.ID)
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, updated)
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
		return cmdutil.FormatOutput(cmd, types)
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
			return cmdutil.FormatOutput(cmd, rows)
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
			return cmdutil.FormatOutput(cmd, backup)
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

// waitForVolumeAvailable polls until the volume reaches "available" status.
func waitForVolumeAvailable(volumeAPI *api.VolumeAPI, id string, wc *cmdutil.WaitConfig) error {
	return cmdutil.WaitFor(*wc, func() (bool, string, error) {
		v, err := volumeAPI.GetVolume(id)
		if err != nil {
			return false, "", err
		}
		if v.Status == "available" {
			return true, v.Status, nil
		}
		if v.Status == "error" {
			return false, v.Status, fmt.Errorf("volume %s entered error state", id)
		}
		return false, v.Status, nil
	})
}
