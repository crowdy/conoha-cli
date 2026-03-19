package output

import "io"

// Formatter formats and writes data to a writer.
type Formatter interface {
	Format(w io.Writer, data any) error
}

// Options configures output formatting behavior.
type Options struct {
	Format    string
	NoHeaders bool
}

// New creates a formatter for the given format name.
func New(format string) Formatter {
	return NewWithOptions(Options{Format: format})
}

// NewWithOptions creates a formatter with the given options.
func NewWithOptions(opts Options) Formatter {
	switch opts.Format {
	case "json":
		return &JSONFormatter{}
	case "yaml":
		return &YAMLFormatter{}
	case "csv":
		return &CSVFormatter{NoHeaders: opts.NoHeaders}
	default:
		return &TableFormatter{NoHeaders: opts.NoHeaders}
	}
}
