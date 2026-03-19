package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/output"
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
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, info)
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
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, containers)
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
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, objects)
	},
}

var cpCmd = &cobra.Command{
	Use: "cp <src> <dst>", Short: "Copy files to/from object storage", Args: cmdutil.ExactArgs(2),
	Long: `Copy files between local filesystem and object storage.
Use container/object format for remote paths.

Examples:
  conoha storage cp myfile.txt mycontainer/myfile.txt    # upload
  conoha storage cp mycontainer/myfile.txt ./myfile.txt  # download`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		storageAPI := api.NewObjectStorageAPI(client)

		src, dst := args[0], args[1]

		// Determine direction: if src contains "/" and doesn't exist locally, it's a download
		if _, err := os.Stat(src); err != nil {
			// Download: src is remote
			container, object := splitPath(src)
			if container == "" || object == "" {
				return fmt.Errorf("remote path must be container/object")
			}
			return storageAPI.DownloadObject(container, object, dst)
		}
		// Upload: src is local
		container, object := splitPath(dst)
		if container == "" {
			return fmt.Errorf("remote path must be container/object")
		}
		if object == "" {
			object = filepath.Base(src)
		}
		return storageAPI.UploadObject(container, object, src)
	},
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
