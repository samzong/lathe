package runtime

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// FormatOutput renders response bytes in the requested format.
// Supported: "table" (default), "json", "yaml", "raw".
// Non-JSON bytes fall back to raw output for table/json/yaml formats.
func FormatOutput(data []byte, format string, w io.Writer, hints OutputHints) error {
	if len(data) == 0 {
		return nil
	}
	switch format {
	case "", "table":
		return renderTable(data, w, hints)
	case "json":
		return renderJSON(data, w)
	case "yaml", "yml":
		return renderYAML(data, w)
	case "raw":
		_, err := w.Write(data)
		return err
	default:
		return fmt.Errorf("unknown output format: %s (supported: table|json|yaml|raw)", format)
	}
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
