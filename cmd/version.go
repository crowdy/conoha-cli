package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const banner = `
  ____                 _   _
 / ___|___  _ __   ___| | | | __ _
| |   / _ \| '_ \ / _ \ |_| |/ _` + "`" + ` |
| |__| (_) | | | | (_) |  _  | (_| |
 \____\___/|_| |_|\___/|_| |_|\__,_|
`

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(banner)
		fmt.Printf("  conoha-cli %s\n", version)
		fmt.Println("  Author:  Tonghyun Kim")
		fmt.Println("  License: Apache-2.0")
		fmt.Println("  Home:    https://github.com/crowdy/conoha-cli")
		fmt.Println()
		fmt.Println("  This is an unofficial tool and is not affiliated with")
		fmt.Println("  or endorsed by ConoHa / GMO Internet, Inc.")
	},
}
