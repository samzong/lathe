package runtime

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func FormatOutput(data []byte, format string, w io.Writer, hints OutputHints) error {
	if len(data) == 0 {
		return nil
	}
	f, ok := formatters[format]
	if !ok {
		return fmt.Errorf("unknown output format: %s (supported: table|json|yaml|raw)", format)
	}
	return f.Format(w, data, hints)
}

func renderJSON(data []byte, w io.Writer) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		_, werr := w.Write(data)
		return werr
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func renderYAML(data []byte, w io.Writer) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		_, werr := w.Write(data)
		return werr
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(v)
}
