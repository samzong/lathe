package runtime

type CommandSpec struct {
	Group       string
	Use         string
	Aliases     []string
	Short       string
	Long        string
	Example     string
	OperationID string
	Hidden      bool
	Deprecated  bool
	Method      string
	PathTpl     string
	Params      []ParamSpec
	RequestBody *RequestBody
	Output      OutputHints
}

type ParamSpec struct {
	Name     string
	Flag     string
	In       string
	GoType   string
	Help     string
	Required bool
}

const (
	InPath     = "path"
	InQuery    = "query"
	InHeader   = "header"
	InFormData = "formData"
)

type RequestBody struct {
	Required  bool
	MediaType string
}

type OutputHints struct {
	ListPath          string
	DefaultColumns    []string
	ResponseMediaType string
}
