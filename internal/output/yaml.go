package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

type YAMLFormatter struct{}

func (f *YAMLFormatter) Format(w io.Writer, data any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return err
	}
	return enc.Close()
}
