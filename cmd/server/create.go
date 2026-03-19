package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

const (
	volumePollInterval = 10 * time.Second
	volumePollTimeout  = 5 * time.Minute
	userDataMaxSize    = 16 * 1024 // 16 KiB
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

func init() {
	createCmd.Flags().String("name", "", "server name (required)")
	createCmd.Flags().String("flavor", "", "flavor ID (interactive if omitted)")
	createCmd.Flags().String("image", "", "image ID (interactive if omitted)")
	createCmd.Flags().String("volume", "", "existing volume ID to use as boot disk")
	createCmd.Flags().String("key-name", "", "SSH key name")
	createCmd.Flags().String("admin-pass", "", "admin password")
	createCmd.Flags().String("user-data", "", "startup script file path")
	createCmd.Flags().String("user-data-raw", "", "startup script string (inline)")
	createCmd.Flags().String("user-data-url", "", "startup script URL (wrapped as #include)")
	_ = createCmd.MarkFlagRequired("name")
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

		// Resolve user_data
		userData, err := resolveUserData(cmd)
		if err != nil {
			return err
		}

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

		if keyName == "" {
			keyName, err = selectKeypair(compute)
			if err != nil {
				return err
			}
		}

		// Warn if Windows flavor with user_data
		if userData != "" && len(flavor.Name) > 2 && flavor.Name[2] == 'w' {
			fmt.Fprintln(os.Stderr, "Warning: startup scripts are not supported on Windows flavors")
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
		req.Server.UserData = userData

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

		// Resolve names for summary
		imageAPI := api.NewImageAPI(client)
		imageName := imageID
		if img, err := imageAPI.GetImage(imageID); err == nil {
			imageName = img.Name
		}

		// Print summary
		fmt.Fprintln(os.Stderr, "=== Server Create Summary ===")
		fmt.Fprintf(os.Stderr, "  Name:     %s\n", name)
		fmt.Fprintf(os.Stderr, "  Flavor:   %s (%d vCPU, %s RAM)\n", flavor.Name, flavor.VCPUs, formatMB(flavor.RAM))
		fmt.Fprintf(os.Stderr, "  Image:    %s\n", imageName)
		if volumeID != "" {
			volAnnotation := "[existing]"
			if created {
				volAnnotation = "[new]"
			}
			volInfo := volumeID[:8]
			if vol, err := volumeAPI.GetVolume(volumeID); err == nil {
				volInfo = fmt.Sprintf("%d GB (%s)", vol.Size, vol.VolumeType)
			}
			fmt.Fprintf(os.Stderr, "  Volume:   %s %s\n", volInfo, volAnnotation)
		}
		if keyName != "" {
			fmt.Fprintf(os.Stderr, "  Key:      %s\n", keyName)
		}
		if adminPass != "" {
			fmt.Fprintln(os.Stderr, "  Password: (set)")
		}
		if userData != "" {
			fmt.Fprintln(os.Stderr, "  Startup:  (set)")
		}

		ok, err := prompt.Confirm("Create this server?")
		if err != nil {
			if created {
				fmt.Fprintf(os.Stderr, "Warning: boot volume %s was created but server creation was cancelled.\n", volumeID)
				fmt.Fprintf(os.Stderr, "You can delete it with: conoha volume delete %s\n", volumeID)
			}
			return err
		}
		if !ok {
			if created {
				fmt.Fprintf(os.Stderr, "Warning: boot volume %s was already created.\n", volumeID)
				fmt.Fprintf(os.Stderr, "You can delete it with: conoha volume delete %s\n", volumeID)
			}
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		server, err := compute.CreateServer(req)
		if err != nil {
			if created {
				fmt.Fprintf(os.Stderr, "Warning: boot volume %s was created but server creation failed.\n", volumeID)
				fmt.Fprintf(os.Stderr, "You can delete it with: conoha volume delete %s\n", volumeID)
			}
			return err
		}
		return cmdutil.FormatOutput(cmd, server)
	},
}

// resolveUserData reads the user_data from the appropriate flag and returns base64-encoded content.
func resolveUserData(cmd *cobra.Command) (string, error) {
	filePath, _ := cmd.Flags().GetString("user-data")
	raw, _ := cmd.Flags().GetString("user-data-raw")
	url, _ := cmd.Flags().GetString("user-data-url")

	// Check mutual exclusion
	count := 0
	if filePath != "" {
		count++
	}
	if raw != "" {
		count++
	}
	if url != "" {
		count++
	}
	if count == 0 {
		return "", nil
	}
	if count > 1 {
		return "", fmt.Errorf("only one of --user-data, --user-data-raw, --user-data-url can be specified")
	}

	var content []byte
	switch {
	case filePath != "":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("reading startup script: %w", err)
		}
		content = data
	case raw != "":
		content = []byte(raw)
	case url != "":
		content = []byte("#include\n" + url + "\n")
	}

	if len(content) > userDataMaxSize {
		return "", fmt.Errorf("startup script too large: %d bytes (max %d bytes)", len(content), userDataMaxSize)
	}

	return base64.StdEncoding.EncodeToString(content), nil
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

func selectKeypair(compute *api.ComputeAPI) (string, error) {
	keypairs, err := compute.ListKeypairs()
	if err != nil {
		return "", err
	}
	if len(keypairs) == 0 {
		return "", nil
	}
	items := make([]prompt.SelectItem, len(keypairs)+1)
	items[0] = prompt.SelectItem{Label: "(none)", Value: ""}
	for i, kp := range keypairs {
		items[i+1] = prompt.SelectItem{
			Label: kp.Name,
			Value: kp.Name,
		}
	}
	return prompt.Select("Select keypair", items)
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
	volName, err := prompt.String("Volume name")
	if err != nil {
		return "", false, err
	}
	if volName == "" {
		return "", false, fmt.Errorf("volume name is required")
	}

	volDesc, err := prompt.String("Volume description (optional)")
	if err != nil {
		return "", false, err
	}

	sizeStr, err := prompt.Select("Volume size", volumeSizeChoices)
	if err != nil {
		return "", false, err
	}
	var sizeGB int
	if _, err := fmt.Sscanf(sizeStr, "%d", &sizeGB); err != nil {
		return "", false, fmt.Errorf("invalid volume size: %w", err)
	}

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
