package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ReadBody reads a request body from a file path. If path is "-", reads from stdin.
func ReadBody(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

// BuildBodyFromSet turns repeated --set key.path=value flags into a JSON
// document. Dotted keys produce nested objects. Value types are inferred:
// "true"/"false" → bool, "null" → null, integer/float strings → number,
// otherwise string. No schema validation — runtime stays schema-agnostic;
// the spec only carries a SchemaRef for future use.
func BuildBodyFromSet(sets []string) ([]byte, error) {
	out := map[string]any{}
	for _, kv := range sets {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			return nil, fmt.Errorf("invalid --set %q (expected key=value)", kv)
		}
		path := kv[:eq]
		if path == "" {
			return nil, fmt.Errorf("invalid --set %q (empty key)", kv)
		}
		if err := setNested(out, strings.Split(path, "."), inferValue(kv[eq+1:])); err != nil {
			return nil, err
		}
	}
	return json.Marshal(out)
}

func setNested(m map[string]any, keys []string, v any) error {
	for i, k := range keys {
		if i == len(keys)-1 {
			m[k] = v
			return nil
		}
		next, ok := m[k]
		if !ok {
			n := map[string]any{}
			m[k] = n
			m = n
			continue
		}
		nm, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("conflicting --set: %s is not an object", strings.Join(keys[:i+1], "."))
		}
		m = nm
	}
	return nil
}

func inferValue(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
