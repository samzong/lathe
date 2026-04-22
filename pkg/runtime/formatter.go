package runtime

import "io"

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

func RegisterFormatter(name string, f Formatter) {
	formatters[name] = f
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
