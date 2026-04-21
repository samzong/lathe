package rawir

type RawModule struct {
	Name       string
	Operations []RawOperation
	Schemas    map[string]*RawSchema
}

type RawOperation struct {
	Group       string
	OperationID string
	Summary     string
	Description string
	Method      string
	Path        string
	Parameters  []RawParameter
	RequestBody *RawRequestBody
	Responses   map[string]*RawResponse
}

type RawParameter struct {
	Name        string
	In          string
	Required    bool
	Type        string
	Description string
}

type RawRequestBody struct {
	Required bool
}

type RawResponse struct {
	Schema *RawSchema
}

type RawSchema struct {
	Ref        string
	Type       string
	Properties map[string]*RawSchema
	Items      *RawSchema
}

const RefPrefix = "#/definitions/"

func Resolve(s *RawSchema, defs map[string]*RawSchema) *RawSchema {
	if s == nil {
		return nil
	}
	if s.Ref == "" {
		return s
	}
	if len(s.Ref) < len(RefPrefix) || s.Ref[:len(RefPrefix)] != RefPrefix {
		return nil
	}
	return defs[s.Ref[len(RefPrefix):]]
}
