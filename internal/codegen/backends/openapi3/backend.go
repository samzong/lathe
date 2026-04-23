package openapi3

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/samzong/lathe/internal/codegen/rawir"
	"github.com/samzong/lathe/internal/sourceconfig"
)

type oas3Doc struct {
	Paths      map[string]*pathItem `json:"paths" yaml:"paths"`
	Components *components          `json:"components,omitempty" yaml:"components,omitempty"`
}

type components struct {
	Schemas map[string]*schemaNode `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

type pathItem struct {
	Parameters []parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Get        *operation  `json:"get,omitempty" yaml:"get,omitempty"`
	Post       *operation  `json:"post,omitempty" yaml:"post,omitempty"`
	Put        *operation  `json:"put,omitempty" yaml:"put,omitempty"`
	Delete     *operation  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch      *operation  `json:"patch,omitempty" yaml:"patch,omitempty"`
}

type operation struct {
	OperationID string              `json:"operationId" yaml:"operationId"`
	Tags        []string            `json:"tags" yaml:"tags"`
	Summary     string              `json:"summary" yaml:"summary"`
	Description string              `json:"description" yaml:"description"`
	Parameters  []parameter         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *requestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]response `json:"responses" yaml:"responses"`
}

type parameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"`
	Required    bool        `json:"required" yaml:"required"`
	Schema      *schemaNode `json:"schema,omitempty" yaml:"schema,omitempty"`
	Description string      `json:"description" yaml:"description"`
}

type requestBody struct {
	Required bool                 `json:"required" yaml:"required"`
	Content  map[string]mediaType `json:"content" yaml:"content"`
}

type mediaType struct {
	Schema *schemaNode `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type response struct {
	Content map[string]mediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type schemaNode struct {
	Ref        string                 `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type       string                 `json:"type,omitempty" yaml:"type,omitempty"`
	Properties map[string]*schemaNode `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items      *schemaNode            `json:"items,omitempty" yaml:"items,omitempty"`
}

const oas3RefPrefix = "#/components/schemas/"

func Parse(src *sourceconfig.Source, syncDir string) (*rawir.RawModule, error) {
	all := &oas3Doc{
		Paths: map[string]*pathItem{},
	}
	for _, rel := range src.OpenAPI3.Files {
		p := filepath.Join(syncDir, rel)
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		var doc oas3Doc
		if err := unmarshalAuto(p, data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		mergeDoc(all, &doc, src.Name, p)
	}
	return toRawIR(src.Name, all), nil
}

func unmarshalAuto(path string, data []byte, v any) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, v)
	default:
		return json.Unmarshal(data, v)
	}
}

func mergeDoc(dst, add *oas3Doc, module, origin string) {
	if add.Components != nil && len(add.Components.Schemas) > 0 {
		if dst.Components == nil {
			dst.Components = &components{Schemas: map[string]*schemaNode{}}
		}
		for k, v := range add.Components.Schemas {
			if existing, exists := dst.Components.Schemas[k]; exists {
				if !sameJSON(existing, v) {
					fmt.Fprintf(os.Stderr, "warn: %s: diverging schema %q in %s (kept first)\n", module, k, origin)
				}
				continue
			}
			dst.Components.Schemas[k] = v
		}
	}
	for path, item := range add.Paths {
		if _, exists := dst.Paths[path]; !exists {
			dst.Paths[path] = item
			continue
		}
		existing := dst.Paths[path]
		if item.Get != nil && existing.Get == nil {
			existing.Get = item.Get
		}
		if item.Post != nil && existing.Post == nil {
			existing.Post = item.Post
		}
		if item.Put != nil && existing.Put == nil {
			existing.Put = item.Put
		}
		if item.Delete != nil && existing.Delete == nil {
			existing.Delete = item.Delete
		}
		if item.Patch != nil && existing.Patch == nil {
			existing.Patch = item.Patch
		}
	}
}

func sameJSON(a, b any) bool {
	ja, err := json.Marshal(a)
	if err != nil {
		return false
	}
	jb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(ja) == string(jb)
}

func toRawIR(name string, doc *oas3Doc) *rawir.RawModule {
	mod := &rawir.RawModule{
		Name:    name,
		Schemas: map[string]*rawir.RawSchema{},
	}
	if doc.Components != nil {
		for k, v := range doc.Components.Schemas {
			mod.Schemas[k] = convertSchema(v)
		}
	}
	for path, item := range doc.Paths {
		pathParams := item.Parameters
		for _, pair := range []struct {
			method string
			op     *operation
		}{
			{"GET", item.Get},
			{"POST", item.Post},
			{"PUT", item.Put},
			{"DELETE", item.Delete},
			{"PATCH", item.Patch},
		} {
			if pair.op == nil {
				continue
			}
			mod.Operations = append(mod.Operations, convertOp(pair.op, pair.method, path, pathParams))
		}
	}
	return mod
}

func convertOp(op *operation, method, path string, pathParams []parameter) rawir.RawOperation {
	out := rawir.RawOperation{
		OperationID: op.OperationID,
		Summary:     op.Summary,
		Description: op.Description,
		Method:      method,
		Path:        path,
		Responses:   map[string]*rawir.RawResponse{},
	}
	if len(op.Tags) > 0 && op.Tags[0] != "" {
		out.Group = op.Tags[0]
	}

	seen := map[string]bool{}
	for _, p := range op.Parameters {
		seen[p.Name] = true
		out.Parameters = append(out.Parameters, convertParam(p))
	}
	for _, p := range pathParams {
		if seen[p.Name] {
			continue
		}
		out.Parameters = append(out.Parameters, convertParam(p))
	}

	if op.RequestBody != nil {
		out.RequestBody = &rawir.RawRequestBody{Required: op.RequestBody.Required}
	}

	for code, resp := range op.Responses {
		rs := &rawir.RawResponse{}
		if mt, ok := resp.Content["application/json"]; ok {
			rs.Schema = convertSchema(mt.Schema)
		}
		out.Responses[code] = rs
	}
	return out
}

func convertParam(p parameter) rawir.RawParameter {
	typ := "string"
	if p.Schema != nil && p.Schema.Type != "" {
		typ = p.Schema.Type
	}
	return rawir.RawParameter{
		Name:        p.Name,
		In:          p.In,
		Required:    p.Required,
		Type:        typ,
		Description: p.Description,
	}
}

func convertSchema(s *schemaNode) *rawir.RawSchema {
	if s == nil {
		return nil
	}
	out := &rawir.RawSchema{
		Type: s.Type,
	}
	if s.Ref != "" {
		if strings.HasPrefix(s.Ref, oas3RefPrefix) {
			out.Ref = rawir.RefPrefix + s.Ref[len(oas3RefPrefix):]
		} else {
			out.Ref = s.Ref
		}
	}
	if len(s.Properties) > 0 {
		out.Properties = make(map[string]*rawir.RawSchema, len(s.Properties))
		for k, v := range s.Properties {
			out.Properties[k] = convertSchema(v)
		}
	}
	if s.Items != nil {
		out.Items = convertSchema(s.Items)
	}
	return out
}
