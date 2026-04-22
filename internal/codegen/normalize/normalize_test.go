package normalize

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/samzong/lathe/internal/codegen/rawir"
	"github.com/samzong/lathe/internal/testutil"
)

// Each case supplies a constructed rawir.RawModule and asserts the
// Normalize output against testdata/<name>.golden.json.
func TestNormalize_Golden(t *testing.T) {
	cases := []struct {
		name  string
		input func() *rawir.RawModule
	}{
		{"minimal-get", minimalGet},
		{"request-body-required", requestBodyRequired},
		{"request-body-optional", requestBodyOptional},
		{"list-response", listResponse},
		{"no-op-id", noOpID},
		{"multiple-methods-same-path", multipleMethodsSamePath},
		{"param-in-header", paramInHeader},
		{"param-in-form-data", paramInFormData},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			specs := Normalize(tc.input())
			data, err := json.MarshalIndent(specs, "", "  ")
			if err != nil {
				t.Fatalf("marshal specs: %v", err)
			}
			data = append(data, '\n')
			testutil.AssertGolden(t, filepath.Join("testdata", tc.name+".golden.json"), data)
		})
	}
}

func minimalGet() *rawir.RawModule {
	return &rawir.RawModule{
		Name: "demo",
		Operations: []rawir.RawOperation{{
			Group:       "Users",
			OperationID: "Users_GetUser",
			Summary:     "Get a user by ID.",
			Method:      "GET",
			Path:        "/users/{id}",
			Parameters: []rawir.RawParameter{
				{Name: "id", In: "path", Required: true, Type: "string"},
				{Name: "limit", In: "query", Required: false, Type: "integer", Description: "Max rows."},
			},
			Responses: map[string]*rawir.RawResponse{},
		}},
	}
}

func requestBodyRequired() *rawir.RawModule {
	return &rawir.RawModule{
		Name: "demo",
		Operations: []rawir.RawOperation{{
			Group:       "Users",
			OperationID: "Users_CreateUser",
			Summary:     "Create a user.",
			Method:      "POST",
			Path:        "/users",
			RequestBody: &rawir.RawRequestBody{Required: true},
			Responses:   map[string]*rawir.RawResponse{},
		}},
	}
}

func requestBodyOptional() *rawir.RawModule {
	return &rawir.RawModule{
		Name: "demo",
		Operations: []rawir.RawOperation{{
			Group:       "Users",
			OperationID: "Users_PatchUser",
			Summary:     "Patch a user.",
			Method:      "PATCH",
			Path:        "/users/{id}",
			Parameters: []rawir.RawParameter{
				{Name: "id", In: "path", Required: true, Type: "string"},
			},
			RequestBody: &rawir.RawRequestBody{Required: false},
			Responses:   map[string]*rawir.RawResponse{},
		}},
	}
}

func listResponse() *rawir.RawModule {
	item := &rawir.RawSchema{
		Type: "object",
		Properties: map[string]*rawir.RawSchema{
			"id":   {Type: "string"},
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
	}
	envelope := &rawir.RawSchema{
		Type: "object",
		Properties: map[string]*rawir.RawSchema{
			"items": {Type: "array", Items: &rawir.RawSchema{Ref: rawir.RefPrefix + "Item"}},
		},
	}
	return &rawir.RawModule{
		Name: "demo",
		Schemas: map[string]*rawir.RawSchema{
			"Item":     item,
			"ItemList": envelope,
		},
		Operations: []rawir.RawOperation{{
			Group:       "Items",
			OperationID: "Items_List",
			Summary:     "List items.",
			Method:      "GET",
			Path:        "/items",
			Responses: map[string]*rawir.RawResponse{
				"200": {Schema: envelope},
			},
		}},
	}
}

func noOpID() *rawir.RawModule {
	return &rawir.RawModule{
		Name: "demo",
		Operations: []rawir.RawOperation{{
			Group:       "Users",
			OperationID: "",
			Method:      "GET",
			Path:        "/users",
			Responses:   map[string]*rawir.RawResponse{},
		}},
	}
}

func multipleMethodsSamePath() *rawir.RawModule {
	return &rawir.RawModule{
		Name: "demo",
		Operations: []rawir.RawOperation{
			{
				Group:       "Resources",
				OperationID: "Resources_List",
				Summary:     "List resources.",
				Method:      "GET",
				Path:        "/resources",
				Responses:   map[string]*rawir.RawResponse{},
			},
			{
				Group:       "Resources",
				OperationID: "Resources_Create",
				Summary:     "Create a resource.",
				Method:      "POST",
				Path:        "/resources",
				RequestBody: &rawir.RawRequestBody{Required: true},
				Responses:   map[string]*rawir.RawResponse{},
			},
		},
	}
}

func paramInHeader() *rawir.RawModule {
	return &rawir.RawModule{
		Name: "demo",
		Operations: []rawir.RawOperation{{
			Group:       "Users",
			OperationID: "Users_GetUser",
			Summary:     "Get a user.",
			Method:      "GET",
			Path:        "/users/{id}",
			Parameters: []rawir.RawParameter{
				{Name: "id", In: "path", Required: true, Type: "string"},
				{Name: "X-Request-Id", In: "header", Required: false, Type: "string", Description: "Trace id."},
			},
			Responses: map[string]*rawir.RawResponse{},
		}},
	}
}

func paramInFormData() *rawir.RawModule {
	return &rawir.RawModule{
		Name: "demo",
		Operations: []rawir.RawOperation{{
			Group:       "Uploads",
			OperationID: "Uploads_Create",
			Summary:     "Upload a file.",
			Method:      "POST",
			Path:        "/uploads",
			Parameters: []rawir.RawParameter{
				{Name: "file", In: "formData", Required: true, Type: "string", Description: "Binary content."},
			},
			Responses: map[string]*rawir.RawResponse{},
		}},
	}
}
