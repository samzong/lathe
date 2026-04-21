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
