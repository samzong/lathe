package runtime

// CommandSpec is the declarative description of a single API operation.
// Code generators emit []CommandSpec; the generic runner (Build) turns it into
// a cobra command tree at startup. This is the IR layer that decouples
// "what the API looks like" from "how the CLI executes it".
type CommandSpec struct {
	Group        string
	Use          string
	Short        string
	Method       string
	PathTpl      string
	Params       []ParamSpec
	HasBody      bool
	BodyRequired bool
	Output       OutputHints
}

// ParamSpec describes a single operation parameter. In is "path" or "query".
// Body parameters are collapsed into CommandSpec.HasBody / BodyRequired.
type ParamSpec struct {
	Name     string
	Flag     string
	In       string
	GoType   string
	Help     string
	Required bool
}

const (
	InPath  = "path"
	InQuery = "query"
)

type OutputHints struct {
	ListPath       string
	DefaultColumns []string
}
