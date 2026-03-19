package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

var Cmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage object storage",
}

func init() {
	Cmd.AddCommand(accountCmd)
	Cmd.AddCommand(containerCmd)
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(cpCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(unpublishCmd)

	cpCmd.Flags().BoolP("recursive", "r", false, "copy directories recursively")
}

var accountCmd = &cobra.Command{
	Use: "account", Short: "Show account info",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		info, err := api.NewObjectStorageAPI(client).GetAccountInfo()
		if err != nil {
			return err
		}
		type row struct {
			ContainerCount int    `json:"container_count"`
			ObjectCount    int    `json:"object_count"`
			BytesUsed      string `json:"bytes_used"`
		}
		r := row{
			ContainerCount: info.ContainerCount,
			ObjectCount:    info.ObjectCount,
			BytesUsed:      cmdutil.FormatBytes(info.BytesUsed),
		}
		return cmdutil.FormatOutput(cmd, r)
	},
}

var containerCmd = &cobra.Command{Use: "container", Short: "Manage containers"}

func init() {
	containerListCmd := &cobra.Command{
		Use: "list", Short: "List containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			containers, err := api.NewObjectStorageAPI(client).ListContainers()
			if err != nil {
				return err
			}
			type row struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
				Size  string `json:"size"`
			}
			rows := make([]row, len(containers))
			for i, c := range containers {
				rows[i] = row{Name: c.Name, Count: c.Count, Size: cmdutil.FormatBytes(c.Bytes)}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}

	containerCreateCmd := &cobra.Command{
		Use: "create <name>", Short: "Create a container", Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewObjectStorageAPI(client).CreateContainer(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Container %s created\n", args[0])
			return nil
		},
	}

	containerDeleteCmd := &cobra.Command{
		Use: "delete <name>", Short: "Delete a container", Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete container %q? All objects will be removed", args[0]))
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
			if err := api.NewObjectStorageAPI(client).DeleteContainer(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Container %s deleted\n", args[0])
			return nil
		},
	}

	containerCmd.AddCommand(containerListCmd, containerCreateCmd, containerDeleteCmd)
}

var lsCmd = &cobra.Command{
	Use: "ls <container>", Short: "List objects in a container", Args: cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		objects, err := api.NewObjectStorageAPI(client).ListObjects(args[0])
		if err != nil {
			return err
		}
		type row struct {
			Name         string `json:"name"`
			ContentType  string `json:"content_type"`
			Size         string `json:"size"`
			LastModified string `json:"last_modified"`
		}
		rows := make([]row, len(objects))
		for i, o := range objects {
			rows[i] = row{Name: o.Name, ContentType: o.ContentType, Size: cmdutil.FormatBytes(o.Bytes), LastModified: o.LastModified}
		}
		return cmdutil.FormatOutput(cmd, rows)
	},
}

var cpCmd = &cobra.Command{
	Use: "cp <src> <dst>", Short: "Copy files to/from object storage", Args: cmdutil.ExactArgs(2),
	Long: `Copy files between local filesystem and object storage.
Use container/object format for remote paths.

Examples:
  conoha storage cp myfile.txt mycontainer/myfile.txt    # upload
  conoha storage cp mycontainer/myfile.txt ./myfile.txt  # download
  conoha storage cp -r ./dir mycontainer/prefix          # recursive upload
  conoha storage cp -r mycontainer/prefix ./dir          # recursive download`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		storageAPI := api.NewObjectStorageAPI(client)
		recursive, _ := cmd.Flags().GetBool("recursive")

		src, dst := args[0], args[1]

		// Determine direction: if src doesn't exist locally, it's a download
		srcInfo, srcErr := os.Stat(src)

		if srcErr != nil {
			// Download: src is remote
			container, prefix := splitPath(src)
			if container == "" {
				return fmt.Errorf("remote path must be container/object")
			}
			if recursive {
				return recursiveDownload(storageAPI, container, prefix, dst)
			}
			if prefix == "" {
				return fmt.Errorf("remote path must be container/object")
			}
			return storageAPI.DownloadObject(container, prefix, dst)
		}

		// Upload: src is local
		container, prefix := splitPath(dst)
		if container == "" {
			return fmt.Errorf("remote path must be container/object")
		}

		if recursive {
			if !srcInfo.IsDir() {
				return fmt.Errorf("source %q is not a directory; use -r with directories", src)
			}
			return recursiveUpload(storageAPI, src, container, prefix)
		}

		object := prefix
		if object == "" {
			object = filepath.Base(src)
		}
		return storageAPI.UploadObject(container, object, src)
	},
}

func recursiveUpload(storageAPI *api.ObjectStorageAPI, localDir, container, prefix string) error {
	var files []string
	err := filepath.WalkDir(localDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory: %w", err)
	}

	var failed int
	for i, path := range files {
		rel, _ := filepath.Rel(localDir, path)
		object := rel
		if prefix != "" {
			object = prefix + "/" + rel
		}
		fmt.Fprintf(os.Stderr, "Copying [%d/%d] %s\n", i+1, len(files), rel)
		if err := storageAPI.UploadObject(container, object, path); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to upload %s: %v\n", rel, err)
			failed++
		}
	}
	fmt.Fprintf(os.Stderr, "Done: %d/%d files uploaded\n", len(files)-failed, len(files))
	if failed > 0 {
		return fmt.Errorf("%d file(s) failed to upload", failed)
	}
	return nil
}

func recursiveDownload(storageAPI *api.ObjectStorageAPI, container, prefix, localDir string) error {
	objects, err := storageAPI.ListObjectsWithPrefix(container, prefix)
	if err != nil {
		return err
	}
	if len(objects) == 0 {
		return fmt.Errorf("no objects found with prefix %q in container %q", prefix, container)
	}

	var failed int
	for i, obj := range objects {
		// Compute relative path from prefix
		rel := obj.Name
		if prefix != "" {
			rel = obj.Name[len(prefix):]
			if len(rel) > 0 && rel[0] == '/' {
				rel = rel[1:]
			}
		}
		if rel == "" {
			continue
		}
		localPath := filepath.Join(localDir, rel)

		// Create parent directories
		if dir := filepath.Dir(localPath); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to create directory %s: %v\n", dir, err)
				failed++
				continue
			}
		}

		fmt.Fprintf(os.Stderr, "Copying [%d/%d] %s\n", i+1, len(objects), rel)
		if err := storageAPI.DownloadObject(container, obj.Name, localPath); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to download %s: %v\n", obj.Name, err)
			failed++
		}
	}
	fmt.Fprintf(os.Stderr, "Done: %d/%d files downloaded\n", len(objects)-failed, len(objects))
	if failed > 0 {
		return fmt.Errorf("%d file(s) failed to download", failed)
	}
	return nil
}

var rmCmd = &cobra.Command{
	Use: "rm <container/object>", Short: "Remove an object", Args: cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete %s?", args[0]))
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
		container, object := splitPath(args[0])
		if container == "" || object == "" {
			return fmt.Errorf("path must be container/object")
		}
		if err := api.NewObjectStorageAPI(client).DeleteObject(container, object); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Deleted %s/%s\n", container, object)
		return nil
	},
}

var publishCmd = &cobra.Command{
	Use: "publish <container>", Short: "Make a container public", Args: cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		if err := api.NewObjectStorageAPI(client).PublishContainer(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Container %s is now public\n", args[0])
		fmt.Fprintf(os.Stderr, "Public URL: https://object-storage.%s.conoha.io/v1/AUTH_%s/%s\n",
			client.Region, client.TenantID, args[0])
		return nil
	},
}

var unpublishCmd = &cobra.Command{
	Use: "unpublish <container>", Short: "Make a container private", Args: cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		if err := api.NewObjectStorageAPI(client).UnpublishContainer(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Container %s is now private\n", args[0])
		return nil
	},
}

func splitPath(s string) (container, object string) {
	for i, c := range s {
		if c == '/' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
