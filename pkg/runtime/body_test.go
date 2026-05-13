package runtime

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBuildBodyFromSet(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want map[string]any
	}{
		{
			name: "nested objects + type inference",
			in:   []string{"spec.replicas=3", "metadata.name=demo", "spec.enabled=true", "spec.weight=0.5", "spec.note=hello"},
			want: map[string]any{
				"spec": map[string]any{
					"replicas": float64(3),
					"enabled":  true,
					"weight":   0.5,
					"note":     "hello",
				},
				"metadata": map[string]any{"name": "demo"},
			},
		},
		{
			name: "null value",
			in:   []string{"a=null"},
			want: map[string]any{"a": nil},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := BuildBodyFromSet(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestBuildBodyFromSet_SetStrKeepsStrings(t *testing.T) {
	raw, err := buildBodyFromSet(
		[]string{"spec.replicas=3", "spec.enabled=true"},
		[]string{"spec.stringReplicas=3", "spec.stringEnabled=true", "metadata.name=demo"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	want := map[string]any{
		"spec": map[string]any{
			"replicas":       float64(3),
			"enabled":        true,
			"stringReplicas": "3",
			"stringEnabled":  "true",
		},
		"metadata": map[string]any{"name": "demo"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestBuildBodyFromSet_ArrayIndex(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want map[string]any
	}{
		{
			name: "simple array",
			in:   []string{"items[0]=a", "items[1]=b"},
			want: map[string]any{"items": []any{"a", "b"}},
		},
		{
			name: "array with type inference",
			in:   []string{"ids[0]=1", "ids[1]=2"},
			want: map[string]any{"ids": []any{float64(1), float64(2)}},
		},
		{
			name: "array of objects",
			in:   []string{"containers[0].name=nginx", "containers[0].image=nginx:latest", "containers[1].name=sidecar"},
			want: map[string]any{
				"containers": []any{
					map[string]any{"name": "nginx", "image": "nginx:latest"},
					map[string]any{"name": "sidecar"},
				},
			},
		},
		{
			name: "nested array under object",
			in:   []string{"spec.ports[0]=8080", "spec.ports[1]=9090"},
			want: map[string]any{
				"spec": map[string]any{
					"ports": []any{float64(8080), float64(9090)},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := BuildBodyFromSet(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestBuildBodyFromSet_Errors(t *testing.T) {
	cases := []struct {
		name string
		in   []string
	}{
		{"missing equals", []string{"foo"}},
		{"empty key", []string{"=value"}},
		{"path conflict", []string{"a=1", "a.b=2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildBodyFromSet(tc.in); err == nil {
				t.Errorf("expected error for %v", tc.in)
			}
		})
	}
}
