package cmdutil

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/output"
)

// FormatOutput applies filter, sort, and formats data to stdout.
func FormatOutput(cmd *cobra.Command, data any) error {
	var err error

	// Apply filters
	filters, _ := cmd.Flags().GetStringArray("filter")
	data, err = output.FilterRows(data, filters)
	if err != nil {
		return err
	}

	// Apply sort
	sortBy, _ := cmd.Flags().GetString("sort-by")
	data, err = output.SortRows(data, sortBy)
	if err != nil {
		return err
	}

	// Format output
	noHeaders, _ := cmd.Flags().GetBool("no-headers")
	opts := output.Options{Format: GetFormat(cmd), NoHeaders: noHeaders}
	return output.NewWithOptions(opts).Format(os.Stdout, data)
}

// FormatBytes formats a byte count as a human-readable string.
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)

	switch {
	case bytes >= tb:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(tb))
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%d KB", bytes/kb)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
