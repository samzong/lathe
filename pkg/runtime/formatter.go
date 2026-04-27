package runtime

import (
	"io"
	"sort"
)

type Formatter interface {
	Format(w io.Writer, data []byte, hints OutputHints) error
}

var formatters = map[string]Formatter{
	"":      tableFormatter{},
	"table": tableFormatter{},
	"json":  jsonFormatter{},
	"yaml":  yamlFormatter{},
	"yml":   yamlFormatter{},
	"raw":   rawFormatter{},
}

var formatterAliases = map[string]string{"yml": "yaml"}

func RegisterFormatter(name string, f Formatter) {
	formatters[name] = f
}

func FormatterNames() []string {
	canonical := []string{"table", "json", "yaml", "raw"}
	seen := map[string]struct{}{}
	names := make([]string, 0, len(formatters))
	for _, name := range canonical {
		if _, ok := formatters[name]; ok {
			names = append(names, name)
			seen[name] = struct{}{}
		}
	}
	extra := make([]string, 0)
	for name := range formatters {
		if name == "" {
			continue
		}
		if _, ok := formatterAliases[name]; ok {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		extra = append(extra, name)
	}
	sort.Strings(extra)
	names = append(names, extra...)
	return names
}

type tableFormatter struct{}

func (tableFormatter) Format(w io.Writer, data []byte, hints OutputHints) error {
	return renderTable(data, w, hints)
}

type jsonFormatter struct{}

func (jsonFormatter) Format(w io.Writer, data []byte, hints OutputHints) error {
	return renderJSON(data, w)
}

type yamlFormatter struct{}

func (yamlFormatter) Format(w io.Writer, data []byte, hints OutputHints) error {
	return renderYAML(data, w)
}

type rawFormatter struct{}

func (rawFormatter) Format(w io.Writer, data []byte, hints OutputHints) error {
	_, err := w.Write(data)
	return err
}
