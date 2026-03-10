package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/output"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

const (
	volumePollInterval = 10 * time.Second
	volumePollTimeout  = 5 * time.Minute
)

// volumeTypeChoices maps user-friendly names to API volume type values.
var volumeTypeChoices = []prompt.SelectItem{
	{Label: "boot-vps-default (c3j1-ds02-boot)", Value: "c3j1-ds02-boot"},
	{Label: "boot-vps-gpu (c3j1-ds03-boot)", Value: "c3j1-ds03-boot"},
	{Label: "boot-game-default (c3j1-ds01-boot)", Value: "c3j1-ds01-boot"},
	{Label: "boot-game-gpu (c3j1-ds03-boot)", Value: "c3j1-ds03-boot"},
}

var volumeSizeChoices = []prompt.SelectItem{
	{Label: "100GB (boot volume)", Value: "100"},
	{Label: "200GB (additional volume)", Value: "200"},
	{Label: "500GB (additional volume)", Value: "500"},
}

// Cmd is the server command group.
var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Manage compute servers",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(renameCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(rebootCmd)
	Cmd.AddCommand(resizeCmd)
	Cmd.AddCommand(rebuildCmd)
	Cmd.AddCommand(consoleCmd)
	Cmd.AddCommand(ipsCmd)
	Cmd.AddCommand(metadataCmd)
	Cmd.AddCommand(attachVolumeCmd)
	Cmd.AddCommand(detachVolumeCmd)

	createCmd.Flags().String("name", "", "server name (required)")
	createCmd.Flags().String("flavor", "", "flavor ID (interactive if omitted)")
	createCmd.Flags().String("image", "", "image ID (interactive if omitted)")
	createCmd.Flags().String("volume", "", "existing volume ID to use as boot disk")
	createCmd.Flags().String("key-name", "", "SSH key name")
	createCmd.Flags().String("admin-pass", "", "admin password")
	_ = createCmd.MarkFlagRequired("name")

	rebootCmd.Flags().Bool("hard", false, "perform hard reboot")
}

func getComputeAPI(cmd *cobra.Command) (*api.ComputeAPI, error) {
	client, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, err
	}
	return api.NewComputeAPI(client), nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		servers, err := compute.ListServers()
		if err != nil {
			return err
		}

		flavors, err := compute.ListFlavors()
		if err != nil {
			return err
		}
		flavorMap := make(map[string]string, len(flavors))
		for _, f := range flavors {
			flavorMap[f.ID] = f.Name
		}

		type serverRow struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Flavor string `json:"flavor"`
			Tag    string `json:"tag"`
		}
		rows := make([]serverRow, len(servers))
		for i, s := range servers {
			flavorName := flavorMap[s.Flavor.ID]
			if flavorName == "" {
				flavorName = s.Flavor.ID
			}
			rows[i] = serverRow{
				ID:     s.ID,
				Name:   s.Name,
				Status: s.Status,
				Flavor: flavorName,
				Tag:    s.Metadata["instance_name_tag"],
			}
		}

		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show server details",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)
		server, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		format := cmdutil.GetFormat(cmd)
		if format != "" && format != "table" {
			return output.New(format).Format(os.Stdout, server)
		}

		// Resolve flavor name
		flavorName := server.Flavor.ID
		if f, err := compute.GetFlavor(server.Flavor.ID); err == nil {
			flavorName = fmt.Sprintf("%s (%s)", f.Name, f.ID)
		}

		// Resolve image name
		imageName := ""
		imageID := server.ImageID
		if imageID == "" && len(server.VolumesAttached) > 0 {
			// Boot from volume: resolve image from the attached volume's metadata
			volumeAPI := api.NewVolumeAPI(client)
			if vol, err := volumeAPI.GetVolume(server.VolumesAttached[0].ID); err == nil {
				if id, ok := vol.VolumeImageMetadata["image_id"]; ok {
					imageID = id
				}
				if name, ok := vol.VolumeImageMetadata["image_name"]; ok && imageID != "" {
					imageName = fmt.Sprintf("%s (%s)", name, imageID)
				}
			}
		}
		if imageName == "" && imageID != "" {
			imageAPI := api.NewImageAPI(client)
			if img, err := imageAPI.GetImage(imageID); err == nil {
				imageName = fmt.Sprintf("%s (%s)", img.Name, img.ID)
			} else {
				imageName = imageID
			}
		}

		printServerDetail(server, flavorName, imageName)
		return nil
	},
}

func printServerDetail(s *model.Server, flavorName, imageName string) {
	jst := time.FixedZone("JST", 9*60*60)

	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Name:      %s\n", s.Name)
	if tag := s.Metadata["instance_name_tag"]; tag != "" {
		fmt.Printf("Name Tag:  %s\n", tag)
	}
	fmt.Printf("Status:    %s\n", s.Status)
	fmt.Printf("Flavor:    %s\n", flavorName)
	fmt.Printf("Image:     %s\n", imageName)
	fmt.Printf("Key Name:  %s\n", s.KeyName)
	fmt.Printf("Tenant:    %s\n", s.TenantID)
	fmt.Printf("Created:   %s (%s JST)\n",
		s.Created.Format(time.RFC3339),
		s.Created.In(jst).Format("2006-01-02 15:04"))
	if !s.Updated.IsZero() {
		fmt.Printf("Updated:   %s (%s JST)\n",
			s.Updated.Format(time.RFC3339),
			s.Updated.In(jst).Format("2006-01-02 15:04"))
	}

	if len(s.Addresses) > 0 {
		fmt.Println("Addresses:")
		for net, addrs := range s.Addresses {
			for _, a := range addrs {
				fmt.Printf("  %s: %s (v%d, %s)\n", net, a.Addr, a.Version, a.Type)
			}
		}
	}

	if len(s.Metadata) > 0 {
		fmt.Println("Metadata:")
		for k, v := range s.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

var renameCmd = &cobra.Command{
	Use:   "rename <id|name> <new-name>",
	Short: "Rename a server",
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		server, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}
		renamed, err := compute.RenameServer(server.ID, args[1])
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server renamed: %s -> %s\n", args[0], renamed.Name)
		return nil
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new server",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)

		name, _ := cmd.Flags().GetString("name")
		flavorID, _ := cmd.Flags().GetString("flavor")
		imageID, _ := cmd.Flags().GetString("image")
		flagVolumeID, _ := cmd.Flags().GetString("volume")
		keyName, _ := cmd.Flags().GetString("key-name")
		adminPass, _ := cmd.Flags().GetString("admin-pass")

		// Resolve flavor (need full struct for volume decision)
		var flavor *model.Flavor
		if flavorID != "" {
			flavor, err = compute.GetFlavor(flavorID)
			if err != nil {
				return fmt.Errorf("flavor %q not found: %w", flavorID, err)
			}
		} else {
			flavor, err = selectFlavor(compute)
			if err != nil {
				return err
			}
		}

		if imageID == "" {
			imageAPI := api.NewImageAPI(client)
			imageID, err = selectImage(imageAPI)
			if err != nil {
				return err
			}
		}

		// Resolve boot volume
		volumeAPI := api.NewVolumeAPI(client)
		volumeID, created, err := resolveBootVolume(volumeAPI, flavor, imageID, flagVolumeID)
		if err != nil {
			return err
		}

		// Build request
		req := &model.ServerCreateRequest{}
		req.Server.Name = name
		req.Server.FlavorRef = flavor.ID
		req.Server.KeyName = keyName
		req.Server.AdminPass = adminPass

		if volumeID != "" {
			// Boot from volume: imageRef must be empty
			req.Server.BlockDeviceMapping = []model.BlockDeviceMapping{
				{
					UUID:                volumeID,
					SourceType:          "volume",
					DestinationType:     "volume",
					BootIndex:           0,
					DeleteOnTermination: false,
				},
			}
		} else {
			// Dedicated flavor: boot from image directly
			req.Server.ImageRef = imageID
		}

		server, err := compute.CreateServer(req)
		if err != nil {
			if created {
				fmt.Fprintf(os.Stderr, "Warning: boot volume %s was created but server creation failed.\n", volumeID)
				fmt.Fprintf(os.Stderr, "You can delete it with: conoha volume delete %s\n", volumeID)
			}
			return err
		}

		fmt.Fprintf(os.Stderr, "Server created: %s (ID: %s)\n", name, server.ID)
		if server.AdminPass != "" {
			fmt.Fprintf(os.Stderr, "Admin password: %s\n", server.AdminPass)
		}

		format := cmdutil.GetFormat(cmd)
		if format != "" && format != "table" {
			return output.New(format).Format(os.Stdout, server)
		}
		return nil
	},
}

func selectFlavor(compute *api.ComputeAPI) (*model.Flavor, error) {
	flavors, err := compute.ListFlavors()
	if err != nil {
		return nil, err
	}
	sort.Slice(flavors, func(i, j int) bool {
		if flavors[i].VCPUs != flavors[j].VCPUs {
			return flavors[i].VCPUs < flavors[j].VCPUs
		}
		return flavors[i].RAM < flavors[j].RAM
	})
	items := make([]prompt.SelectItem, len(flavors))
	flavorMap := make(map[string]*model.Flavor, len(flavors))
	for i, f := range flavors {
		items[i] = prompt.SelectItem{
			Label: fmt.Sprintf("%s (%d vCPU, %s RAM)", f.Name, f.VCPUs, formatMB(f.RAM)),
			Value: f.ID,
		}
		flavorMap[f.ID] = &flavors[i]
	}
	id, err := prompt.Select("Select flavor", items)
	if err != nil {
		return nil, err
	}
	return flavorMap[id], nil
}

// flavorNeedsVolume returns true if the flavor requires a boot volume.
// Flavor naming: g2l-xxx (Linux), g2w-xxx (Windows) need volumes; g2d-xxx (dedicated) does not.
func flavorNeedsVolume(flavorName string) bool {
	return len(flavorName) > 2 && flavorName[2] != 'd'
}

// resolveBootVolume determines the boot volume for server creation.
// Returns volumeID (empty if not needed), whether a new volume was created, and any error.
func resolveBootVolume(volumeAPI *api.VolumeAPI, flavor *model.Flavor, imageID string, flagVolumeID string) (string, bool, error) {
	if !flavorNeedsVolume(flavor.Name) {
		return "", false, nil
	}

	// --volume flag specified
	if flagVolumeID != "" {
		vol, err := volumeAPI.GetVolume(flagVolumeID)
		if err != nil {
			return "", false, fmt.Errorf("volume %q not found: %w", flagVolumeID, err)
		}
		if vol.Status != "available" {
			return "", false, fmt.Errorf("volume %s is not available (status: %s)", flagVolumeID, vol.Status)
		}
		return flagVolumeID, false, nil
	}

	// Interactive selection
	items := []prompt.SelectItem{
		{Label: "Create new volume", Value: "new"},
		{Label: "Use existing volume", Value: "existing"},
	}
	choice, err := prompt.Select("Boot volume", items)
	if err != nil {
		return "", false, err
	}

	if choice == "new" {
		return createBootVolume(volumeAPI, imageID)
	}
	return selectExistingVolume(volumeAPI)
}

func createBootVolume(volumeAPI *api.VolumeAPI, imageID string) (string, bool, error) {
	// Prompt for volume name
	volName, err := prompt.String("Volume name")
	if err != nil {
		return "", false, err
	}
	if volName == "" {
		return "", false, fmt.Errorf("volume name is required")
	}

	// Prompt for description (optional)
	volDesc, err := prompt.String("Volume description (optional)")
	if err != nil {
		return "", false, err
	}

	// Prompt for size
	sizeStr, err := prompt.Select("Volume size", volumeSizeChoices)
	if err != nil {
		return "", false, err
	}
	var sizeGB int
	if _, err := fmt.Sscanf(sizeStr, "%d", &sizeGB); err != nil {
		return "", false, fmt.Errorf("invalid volume size: %w", err)
	}

	// Prompt for volume type
	volType, err := prompt.Select("Volume type", volumeTypeChoices)
	if err != nil {
		return "", false, err
	}

	fmt.Fprintf(os.Stderr, "Creating boot volume %q (%dGB, %s)...\n", volName, sizeGB, volType)
	req := &model.VolumeCreateRequest{}
	req.Volume.Size = sizeGB
	req.Volume.Name = volName
	req.Volume.Description = volDesc
	req.Volume.VolumeType = volType
	req.Volume.ImageRef = imageID
	vol, err := volumeAPI.CreateVolume(req)
	if err != nil {
		return "", false, fmt.Errorf("creating boot volume: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Waiting for volume %s to become available...\n", vol.ID)
	vol, err = waitForVolume(volumeAPI, vol.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: boot volume %s was created but may not be ready.\n", vol.ID)
		fmt.Fprintf(os.Stderr, "You can delete it with: conoha volume delete %s\n", vol.ID)
		return "", true, err
	}
	fmt.Fprintf(os.Stderr, "Volume %s is ready.\n", vol.ID)
	return vol.ID, true, nil
}

func selectExistingVolume(volumeAPI *api.VolumeAPI) (string, bool, error) {
	volumes, err := volumeAPI.ListVolumes()
	if err != nil {
		return "", false, err
	}
	var available []model.Volume
	for _, v := range volumes {
		if v.Status == "available" {
			available = append(available, v)
		}
	}
	if len(available) == 0 {
		return "", false, fmt.Errorf("no available volumes found; create one first with: conoha volume create")
	}
	items := make([]prompt.SelectItem, len(available))
	for i, v := range available {
		label := fmt.Sprintf("%s (%dGB, %s)", v.Name, v.Size, v.Status)
		if v.Name == "" {
			label = fmt.Sprintf("%s (%dGB, %s)", v.ID[:8], v.Size, v.Status)
		}
		items[i] = prompt.SelectItem{
			Label: label,
			Value: v.ID,
		}
	}
	id, err := prompt.Select("Select volume", items)
	if err != nil {
		return "", false, err
	}
	return id, false, nil
}

// waitForVolume polls until the volume reaches "available" status.
// On Ctrl+C, it warns that volume creation continues server-side.
func waitForVolume(volumeAPI *api.VolumeAPI, id string) (*model.Volume, error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	deadline := time.Now().Add(volumePollTimeout)
	for {
		vol, err := volumeAPI.GetVolume(id)
		if err != nil {
			return nil, fmt.Errorf("checking volume status: %w", err)
		}
		if vol.Status == "available" {
			return vol, nil
		}
		if vol.Status == "error" {
			return vol, fmt.Errorf("volume %s entered error state", id)
		}
		if time.Now().After(deadline) {
			return vol, fmt.Errorf("timeout waiting for volume %s (status: %s)", id, vol.Status)
		}
		fmt.Fprintf(os.Stderr, "  volume %s status: %s\n", id, vol.Status)

		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "\nInterrupted. Volume creation is still in progress on the server.\n")
			fmt.Fprintf(os.Stderr, "Check status with: conoha volume show %s\n", id)
			return vol, fmt.Errorf("interrupted while waiting for volume %s", id)
		case <-time.After(volumePollInterval):
		}
	}
}

func selectImage(imageAPI *api.ImageAPI) (string, error) {
	images, err := imageAPI.ListImages()
	if err != nil {
		return "", err
	}
	var active []model.Image
	for _, img := range images {
		if img.Status == "active" {
			active = append(active, img)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].Name < active[j].Name
	})
	items := make([]prompt.SelectItem, len(active))
	for i, img := range active {
		items[i] = prompt.SelectItem{
			Label: img.Name,
			Value: img.ID,
		}
	}
	return prompt.Select("Select image", items)
}

func formatMB(mb int) string {
	if mb == 0 {
		return "0"
	}
	if mb%1024 == 0 {
		return fmt.Sprintf("%dG", mb/1024)
	}
	return fmt.Sprintf("%dM", mb)
}

// resolveServerID resolves an id-or-name argument to a server ID.
func resolveServerID(compute *api.ComputeAPI, idOrName string) (string, error) {
	s, err := compute.FindServer(idOrName)
	if err != nil {
		return "", err
	}
	return s.ID, nil
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id|name>",
	Short: "Delete a server",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.DeleteServer(id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s deleted\n", args[0])
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <id|name>",
	Short: "Start a server",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.StartServer(id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s starting\n", args[0])
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <id|name>",
	Short: "Stop a server",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.StopServer(id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s stopping\n", args[0])
		return nil
	},
}

var rebootCmd = &cobra.Command{
	Use:   "reboot <id|name>",
	Short: "Reboot a server",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		hard, _ := cmd.Flags().GetBool("hard")
		if err := compute.RebootServer(id, hard); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s rebooting\n", args[0])
		return nil
	},
}

var resizeCmd = &cobra.Command{
	Use:   "resize <id|name> <flavor-id>",
	Short: "Resize a server",
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.ResizeServer(id, args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s resizing to flavor %s\n", args[0], args[1])
		return nil
	},
}

var rebuildCmd = &cobra.Command{
	Use:   "rebuild <id|name> <image-id>",
	Short: "Rebuild a server with a new image",
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.RebuildServer(id, args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s rebuilding with image %s\n", args[0], args[1])
		return nil
	},
}

var consoleCmd = &cobra.Command{
	Use:   "console <id|name>",
	Short: "Get VNC console URL",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		resp, err := compute.GetConsole(id)
		if err != nil {
			return err
		}
		fmt.Println(resp.RemoteConsole.URL)
		return nil
	},
}

var ipsCmd = &cobra.Command{
	Use:   "ips <id|name>",
	Short: "Show server IP addresses",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		server, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		format := cmdutil.GetFormat(cmd)
		if format != "" && format != "table" {
			return output.New(format).Format(os.Stdout, server.Addresses)
		}

		for net, addrs := range server.Addresses {
			for _, a := range addrs {
				fmt.Printf("%s: %s (v%d, %s)\n", net, a.Addr, a.Version, a.Type)
			}
		}
		return nil
	},
}

var metadataCmd = &cobra.Command{
	Use:   "metadata <id|name>",
	Short: "Show server metadata",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		meta, err := compute.GetServerMetadata(id)
		if err != nil {
			return err
		}

		format := cmdutil.GetFormat(cmd)
		if format != "" && format != "table" {
			return output.New(format).Format(os.Stdout, meta)
		}

		for k, v := range meta {
			fmt.Printf("%s: %s\n", k, v)
		}
		return nil
	},
}

var attachVolumeCmd = &cobra.Command{
	Use:   "attach-volume <server-id> <volume-id>",
	Short: "Attach a volume to a server",
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.AttachVolume(args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Volume %s attached to server %s\n", args[1], args[0])
		return nil
	},
}

var detachVolumeCmd = &cobra.Command{
	Use:   "detach-volume <server-id> <volume-id>",
	Short: "Detach a volume from a server",
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.DetachVolume(args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Volume %s detached from server %s\n", args[1], args[0])
		return nil
	},
}
