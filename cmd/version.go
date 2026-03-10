package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("conoha version %s by crowdy@gmail.com\n", version)
		fmt.Println("This is an unofficial tool and is not affiliated with or endorsed by ConoHa/GMO Internet Group.")
	},
}
