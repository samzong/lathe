package sourceconfig

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	BackendSwagger = "swagger"
	BackendProto   = "proto"
)

type Config struct {
	Sources map[string]*Source `yaml:"sources"`
}

type Source struct {
	Name      string         `yaml:"-"`
	RepoURL   string         `yaml:"repo_url"`
	PinnedTag string         `yaml:"pinned_tag"`
	Backend   string         `yaml:"backend"`
	Swagger   *SwaggerConfig `yaml:"swagger,omitempty"`
	Proto     *ProtoConfig   `yaml:"proto,omitempty"`
}

type SwaggerConfig struct {
	Files []string `yaml:"files"`
}

type ProtoConfig struct {
	Staging     []StagingEntry `yaml:"staging"`
	Entries     []string       `yaml:"entries"`
	ImportRoots []string       `yaml:"import_roots,omitempty"`
}

type StagingEntry struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("%s declares no sources", path)
	}
	for name, src := range cfg.Sources {
		src.Name = name
		if err := validate(src); err != nil {
			return nil, fmt.Errorf("source %q: %w", name, err)
		}
	}
	return &cfg, nil
}

func (c *Config) Ordered() []*Source {
	names := make([]string, 0, len(c.Sources))
	for n := range c.Sources {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]*Source, 0, len(names))
	for _, n := range names {
		out = append(out, c.Sources[n])
	}
	return out
}

func validate(s *Source) error {
	if s.RepoURL == "" {
		return fmt.Errorf("missing repo_url")
	}
	if s.PinnedTag == "" {
		return fmt.Errorf("missing pinned_tag")
	}
	switch s.Backend {
	case BackendSwagger:
		if s.Swagger == nil || len(s.Swagger.Files) == 0 {
			return fmt.Errorf("backend=swagger requires non-empty swagger.files")
		}
		if s.Proto != nil {
			return fmt.Errorf("backend=swagger must not set proto block")
		}
	case BackendProto:
		if s.Proto == nil || len(s.Proto.Entries) == 0 {
			return fmt.Errorf("backend=proto requires non-empty proto.entries")
		}
		if len(s.Proto.Staging) == 0 {
			return fmt.Errorf("backend=proto requires non-empty proto.staging")
		}
		if s.Swagger != nil {
			return fmt.Errorf("backend=proto must not set swagger block")
		}
	case "":
		return fmt.Errorf("missing backend")
	default:
		return fmt.Errorf("unknown backend %q", s.Backend)
	}
	return nil
}
