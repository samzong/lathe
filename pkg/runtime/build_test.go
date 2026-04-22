package runtime

import (
	"testing"

	"github.com/spf13/cobra"
)

func newRootWithModuleGroup() *cobra.Command {
	root := &cobra.Command{Use: "myctl"}
	root.AddGroup(&cobra.Group{ID: ModuleGroupID, Title: "Modules"})
	return root
}

func TestBuild_PopulatesGroupAndOpTree(t *testing.T) {
	specs := []CommandSpec{
		{
			Group:   "Users",
			Use:     "get-user",
			Short:   "Get a user",
			Method:  "GET",
			PathTpl: "/users/{id}",
			Params: []ParamSpec{
				{Name: "id", Flag: "id", In: InPath, GoType: "string", Required: true, Help: "User id"},
				{Name: "limit", Flag: "limit", In: InQuery, GoType: "int64", Help: "Page size"},
			},
		},
		{
			Group:   "Items",
			Use:     "list-items",
			Short:   "List items",
			Method:  "GET",
			PathTpl: "/items",
			Params: []ParamSpec{
				{Name: "verbose", Flag: "verbose", In: InQuery, GoType: "bool", Help: "Verbose output"},
			},
		},
	}

	root := newRootWithModuleGroup()
	Build(root, "demo", specs)

	svc := mustFindChild(t, root, "demo")
	usersGroup := mustFindChild(t, svc, "users")
	itemsGroup := mustFindChild(t, svc, "items")

	if len(usersGroup.Commands()) != 1 || usersGroup.Commands()[0].Use != "get-user" {
		t.Errorf("users group commands = %v, want [get-user]", cmdNames(usersGroup.Commands()))
	}
	if len(itemsGroup.Commands()) != 1 || itemsGroup.Commands()[0].Use != "list-items" {
		t.Errorf("items group commands = %v, want [list-items]", cmdNames(itemsGroup.Commands()))
	}

	getUser := usersGroup.Commands()[0]
	if f := getUser.Flag("id"); f == nil {
		t.Errorf("get-user missing --id flag")
	} else if !isRequiredFlag(f.Annotations) {
		t.Errorf("get-user --id flag is not marked required")
	}
	if f := getUser.Flag("limit"); f == nil {
		t.Errorf("get-user missing --limit flag")
	} else if f.Value.Type() != "int64" {
		t.Errorf("get-user --limit type = %q, want int64", f.Value.Type())
	}

	listItems := itemsGroup.Commands()[0]
	if f := listItems.Flag("verbose"); f == nil {
		t.Errorf("list-items missing --verbose flag")
	} else if f.Value.Type() != "bool" {
		t.Errorf("list-items --verbose type = %q, want bool", f.Value.Type())
	}
}

func TestBuild_EmptySpecsMountsEmptyService(t *testing.T) {
	root := newRootWithModuleGroup()
	Build(root, "demo", nil)

	svc := mustFindChild(t, root, "demo")
	if len(svc.Commands()) != 0 {
		t.Errorf("empty specs should yield no subcommands under demo; got %v", cmdNames(svc.Commands()))
	}
}

func TestBuild_BodyFlagsAttachedWhenHasBody(t *testing.T) {
	specs := []CommandSpec{{
		Group:        "Users",
		Use:          "create-user",
		Method:       "POST",
		PathTpl:      "/users",
		HasBody:      true,
		BodyRequired: true,
	}}

	root := newRootWithModuleGroup()
	Build(root, "demo", specs)

	svc := mustFindChild(t, root, "demo")
	users := mustFindChild(t, svc, "users")
	createUser := mustFindChild(t, users, "create-user")

	for _, name := range []string{"file", "set"} {
		if createUser.Flag(name) == nil {
			t.Errorf("create-user missing --%s flag", name)
		}
	}
}

func mustFindChild(t *testing.T, parent *cobra.Command, use string) *cobra.Command {
	t.Helper()
	for _, c := range parent.Commands() {
		if c.Use == use {
			return c
		}
	}
	t.Fatalf("%s has no child %q; children = %v", parent.Use, use, cmdNames(parent.Commands()))
	return nil
}

func cmdNames(cmds []*cobra.Command) []string {
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.Use)
	}
	return names
}

func isRequiredFlag(annotations map[string][]string) bool {
	for _, v := range annotations[cobra.BashCompOneRequiredFlag] {
		if v == "true" {
			return true
		}
	}
	return false
}
