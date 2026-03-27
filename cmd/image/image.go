package image

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

var Cmd = &cobra.Command{
	Use:   "image",
	Short: "Manage images",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(uploadCmd)
	Cmd.AddCommand(importCmd)

	createCmd.Flags().String("name", "", "image name (required)")
	createCmd.Flags().String("disk-format", "iso", "disk format")
	createCmd.Flags().String("container-format", "bare", "container format")
	_ = createCmd.MarkFlagRequired("name")

	uploadCmd.Flags().String("file", "", "path to image file (required)")
	_ = uploadCmd.MarkFlagRequired("file")

	importCmd.Flags().String("name", "", "image name (required)")
	importCmd.Flags().String("disk-format", "iso", "disk format")
	importCmd.Flags().String("container-format", "bare", "container format")
	importCmd.Flags().String("file", "", "path to image file (required)")
	_ = importCmd.MarkFlagRequired("name")
	_ = importCmd.MarkFlagRequired("file")
	cmdutil.AddWaitFlags(importCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an image record",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		diskFormat, _ := cmd.Flags().GetString("disk-format")
		containerFormat, _ := cmd.Flags().GetString("container-format")

		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		img, err := api.NewImageAPI(client).CreateImage(name, diskFormat, containerFormat)
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, img)
	},
}

var uploadCmd = &cobra.Command{
	Use:   "upload <id>",
	Short: "Upload image file",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return fmt.Errorf("reading file info: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Uploading %s (%s)...\n", filePath, cmdutil.FormatBytes(stat.Size()))

		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		imageAPI := api.NewImageAPI(client)
		if err := imageAPI.UploadImageFile(args[0], f, stat.Size()); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Upload complete.")
		img, err := imageAPI.GetImage(args[0])
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, img)
	},
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Create image record and upload file",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		diskFormat, _ := cmd.Flags().GetString("disk-format")
		containerFormat, _ := cmd.Flags().GetString("container-format")
		filePath, _ := cmd.Flags().GetString("file")

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return fmt.Errorf("reading file info: %w", err)
		}

		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		imageAPI := api.NewImageAPI(client)

		img, err := imageAPI.CreateImage(name, diskFormat, containerFormat)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Image record created: %s\n", img.ID)

		fmt.Fprintf(os.Stderr, "Uploading %s (%s)...\n", filePath, cmdutil.FormatBytes(stat.Size()))
		if err := imageAPI.UploadImageFile(img.ID, f, stat.Size()); err != nil {
			fmt.Fprintf(os.Stderr, "Image record created (ID: %s) but upload failed.\n", img.ID)
			fmt.Fprintf(os.Stderr, "Retry with: conoha image upload %s --file %s\n", img.ID, filePath)
			return err
		}
		fmt.Fprintln(os.Stderr, "Upload complete.")

		if wc := cmdutil.GetWaitConfig(cmd, "image "+name); wc != nil {
			fmt.Fprintf(os.Stderr, "Waiting for image to become active...\n")
			if err := cmdutil.WaitFor(*wc, func() (bool, string, error) {
				current, err := imageAPI.GetImage(img.ID)
				if err != nil {
					return false, "", err
				}
				status := current.Status
				if status == "active" {
					return true, status, nil
				}
				if status == "killed" || status == "deactivated" {
					return false, status, fmt.Errorf("image %s entered %s state", img.ID, status)
				}
				return false, status, nil
			}); err != nil {
				return err
			}
		}

		img, err = imageAPI.GetImage(img.ID)
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, img)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List images",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		images, err := api.NewImageAPI(client).ListImages()
		if err != nil {
			return err
		}

		type row struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Status     string `json:"status"`
			MinDisk    int    `json:"min_disk"`
			Visibility string `json:"visibility"`
		}
		rows := make([]row, len(images))
		for i, img := range images {
			rows[i] = row{ID: img.ID, Name: img.Name, Status: img.Status, MinDisk: img.MinDisk, Visibility: img.Visibility}
		}
		return cmdutil.FormatOutput(cmd, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show image details",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		img, err := api.NewImageAPI(client).GetImage(args[0])
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, img)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an image",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete image %s?", args[0]))
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
		if err := api.NewImageAPI(client).DeleteImage(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Image %s deleted\n", args[0])
		return nil
	},
}
