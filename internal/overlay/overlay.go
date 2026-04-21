package overlay

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var fsys embed.FS

// Override is the per-command hand-written UX patch applied on top of the
// mechanically generated command. Every field is optional; empty means
// "keep whatever the generated layer produced".
type Override struct {
	Aliases []string `yaml:"aliases"`
	Short   string   `yaml:"short"`
	Long    string   `yaml:"long"`
	Example string   `yaml:"example"`
}

type moduleFile struct {
	Commands map[string]Override `yaml:"commands"`
}

var all = map[string]map[string]Override{}

func init() {
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		data, rerr := fsys.ReadFile(path)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "overlay: read %s: %v\n", path, rerr)
			return nil
		}
		var mf moduleFile
		if yerr := yaml.Unmarshal(data, &mf); yerr != nil {
			fmt.Fprintf(os.Stderr, "overlay: parse %s: %v\n", path, yerr)
			return nil
		}
		module := strings.TrimSuffix(path, ".yaml")
		all[module] = mf.Commands
		return nil
	})
}

// Apply patches cmd with the overlay registered for (module, use) if any.
// Fields left empty in the overlay keep the generated values.
func Apply(cmd *cobra.Command, module, use string) {
	m, ok := all[module]
	if !ok {
		return
	}
	o, ok := m[use]
	if !ok {
		return
	}
	if o.Short != "" {
		cmd.Short = o.Short
	}
	if o.Long != "" {
		cmd.Long = o.Long
	}
	if o.Example != "" {
		cmd.Example = o.Example
	}
	if len(o.Aliases) > 0 {
		cmd.Aliases = append(cmd.Aliases, o.Aliases...)
	}
}
