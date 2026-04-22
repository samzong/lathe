package runtime

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// ModuleGroupID is the cobra group ID every generated service command tree
// attaches to. Root command must AddGroup this ID for help output to
// segregate modules from core commands (e.g. auth) and from completion/help.
const ModuleGroupID = "modules"

func AssertSchema(generated int) {
	if generated != SchemaVersion {
		panic(fmt.Sprintf(
			"lathe schema mismatch: generated code uses schema %d but runtime expects %d — re-run codegen",
			generated, SchemaVersion,
		))
	}
}

// Build mounts a service command tree under root, driven entirely by specs.
// Replaces the previous per-operation function approach: every operation is
// data, the execution path is a single function.
func Build(root *cobra.Command, service string, specs []CommandSpec) {
	svc := &cobra.Command{Use: service, Short: service + " API", GroupID: ModuleGroupID}
	groups := map[string]*cobra.Command{}
	for i := range specs {
		s := specs[i]
		g, ok := groups[s.Group]
		if !ok {
			g = &cobra.Command{Use: strings.ToLower(s.Group), Short: s.Group + " operations"}
			groups[s.Group] = g
			svc.AddCommand(g)
		}
		c := buildCmd(s)
		g.AddCommand(c)
	}
	root.AddCommand(svc)
}

func buildCmd(s CommandSpec) *cobra.Command {
	vals := make(map[string]any, len(s.Params))
	var bodyFile string
	var bodySets []string

	cmd := &cobra.Command{
		Use:     s.Use,
		Aliases: s.Aliases,
		Short:   s.Short,
		Long:    s.Long,
		Example: s.Example,
		RunE: func(cmd *cobra.Command, _ []string) error {
			hostname, clientOpts, err := LoadHostOptions(cmd)
			if err != nil {
				return err
			}

			path := s.PathTpl
			q := url.Values{}
			hdrs := map[string]string{}
			form := url.Values{}
			for _, p := range s.Params {
				switch p.In {
				case InPath:
					v := vals[p.Name].(*string)
					path = strings.Replace(path, "{"+p.Name+"}", url.PathEscape(*v), 1)
					continue
				case InHeader:
					if !cmd.Flags().Changed(p.Flag) {
						continue
					}
					hdrs[p.Name] = *vals[p.Name].(*string)
					continue
				case InFormData:
					if !cmd.Flags().Changed(p.Flag) {
						continue
					}
					switch v := vals[p.Name].(type) {
					case *int64:
						form.Set(p.Name, strconv.FormatInt(*v, 10))
					case *bool:
						form.Set(p.Name, strconv.FormatBool(*v))
					case *string:
						form.Set(p.Name, *v)
					}
					continue
				}
				if !cmd.Flags().Changed(p.Flag) {
					continue
				}
				switch v := vals[p.Name].(type) {
				case *int64:
					q.Set(p.Name, strconv.FormatInt(*v, 10))
				case *bool:
					q.Set(p.Name, strconv.FormatBool(*v))
				case *[]string:
					for _, vv := range *v {
						q.Add(p.Name, vv)
					}
				case *string:
					q.Set(p.Name, *v)
				}
			}
			if enc := q.Encode(); enc != "" {
				path = path + "?" + enc
			}

			var body any
			if len(form) > 0 {
				body = form
			} else if s.RequestBody != nil {
				switch {
				case cmd.Flags().Changed("set"):
					raw, berr := BuildBodyFromSet(bodySets)
					if berr != nil {
						return berr
					}
					body = raw
				case cmd.Flags().Changed("file"):
					raw, rerr := ReadBody(bodyFile)
					if rerr != nil {
						return rerr
					}
					body = raw
				case s.RequestBody.Required:
					return fmt.Errorf("request body required: pass --file or --set")
				}
			}

			if v, err := cmd.Root().PersistentFlags().GetBool("debug"); err == nil && v {
				clientOpts.Debug = true
			}
			clientOpts.Headers = hdrs
			data, err := DoRaw(cmd.Context(), hostname, s.Method, path, body, clientOpts)
			if err != nil {
				return err
			}
			format, _ := cmd.Root().PersistentFlags().GetString("output")
			return FormatOutput(data, format, os.Stdout, s.Output)
		},
	}

	for i := range s.Params {
		p := s.Params[i]
		if p.In == InPath {
			v := new(string)
			vals[p.Name] = v
			cmd.Flags().StringVar(v, p.Flag, "", p.Help)
			_ = cmd.MarkFlagRequired(p.Flag)
			continue
		}
		switch p.GoType {
		case "int64":
			v := new(int64)
			vals[p.Name] = v
			cmd.Flags().Int64Var(v, p.Flag, 0, p.Help)
		case "bool":
			v := new(bool)
			vals[p.Name] = v
			cmd.Flags().BoolVar(v, p.Flag, false, p.Help)
		case "[]string":
			v := new([]string)
			vals[p.Name] = v
			cmd.Flags().StringSliceVar(v, p.Flag, nil, p.Help)
		default:
			v := new(string)
			vals[p.Name] = v
			cmd.Flags().StringVar(v, p.Flag, "", p.Help)
		}
		if p.Required {
			_ = cmd.MarkFlagRequired(p.Flag)
		}
	}
	if s.RequestBody != nil {
		fileHelp := "path to JSON body file, or '-' for stdin"
		setHelp := "set body field, e.g. --set spec.replicas=3 (repeatable; nested via dots)"
		if s.RequestBody.Required {
			fileHelp += " (use --file or --set)"
			setHelp += " (use --file or --set)"
		}
		cmd.Flags().StringVarP(&bodyFile, "file", "f", "", fileHelp)
		cmd.Flags().StringSliceVar(&bodySets, "set", nil, setHelp)
	}
	cmd.Hidden = s.Hidden
	if s.Deprecated {
		cmd.Deprecated = "this command is deprecated"
	}
	return cmd
}
