package runtime

const SchemaVersion = 3

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
	Name       string
	Flag       string
	In         string
	GoType     string
	Help       string
	Required   bool
	Default    string
	Enum       []string
	Format     string
	Deprecated bool
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
	Pagination        *PaginationHint
	Streaming         *StreamingHint
}

type PaginationHint struct {
	Strategy   string
	TokenParam string
	TokenField string
	LimitParam string
}

type StreamingHint struct {
	Strategy string
}
