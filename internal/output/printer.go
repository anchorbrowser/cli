package output

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

const (
	FormatJSON = "json"
	FormatYAML = "yaml"
)

// Printer writes structured values in json or yaml.
type Printer struct {
	Format  string
	Compact bool
	Writer  io.Writer
}

func (p Printer) Print(v any) error {
	if p.Writer == nil {
		return fmt.Errorf("output writer is nil")
	}
	switch p.Format {
	case "", FormatJSON:
		var data []byte
		var err error
		if p.Compact {
			data, err = json.Marshal(v)
		} else {
			data, err = json.MarshalIndent(v, "", "  ")
		}
		if err != nil {
			return fmt.Errorf("marshal json output: %w", err)
		}
		_, err = fmt.Fprintln(p.Writer, string(data))
		return err
	case FormatYAML:
		data, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal yaml output: %w", err)
		}
		_, err = fmt.Fprint(p.Writer, string(data))
		return err
	default:
		return fmt.Errorf("unsupported output format %q", p.Format)
	}
}

func ValidateFormat(format string) error {
	switch format {
	case "", FormatJSON, FormatYAML:
		return nil
	default:
		return fmt.Errorf("unsupported output format %q (allowed: json, yaml)", format)
	}
}
