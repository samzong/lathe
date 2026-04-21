package runtime

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/samzong/lathe/internal/config"
)

// ErrNotAuthenticated is the sentinel returned when no host is configured
// and none is selected via --hostname / $<HostEnv>. main.go maps this to
// exit code 4 via errors.Is. The user-facing wording is rendered by
// NewNotAuthenticatedError using the bound manifest.
var ErrNotAuthenticated = errors.New("not authenticated")

// NewNotAuthenticatedError renders the "no host configured" message using
// the bound manifest and wraps ErrNotAuthenticated so errors.Is still works.
func NewNotAuthenticatedError() error {
	name := config.Active().CLI.Name
	return fmt.Errorf("not logged in to any %s host; run `%s auth login` to authenticate: %w", name, name, ErrNotAuthenticated)
}

func ResolveHost(cmd *cobra.Command) (string, error) {
	cli := config.Active().CLI
	h, _ := cmd.Root().PersistentFlags().GetString("hostname")
	if h == "" {
		h = os.Getenv(cli.HostEnv)
	}
	if h != "" {
		return config.NormalizeHostname(h), nil
	}
	hosts, err := config.LoadHosts()
	if err != nil {
		return "", err
	}
	names := hosts.Names()
	switch len(names) {
	case 0:
		return "", NewNotAuthenticatedError()
	case 1:
		return names[0], nil
	default:
		return "", fmt.Errorf("multiple hosts configured (%v); specify --hostname or $%s", names, cli.HostEnv)
	}
}

// LoadHostOptions resolves hostname, token, and client options for the
// current command in one call. The persistent --insecure flag forces insecure
// when set; otherwise the host record's persisted Insecure value applies.
func LoadHostOptions(cmd *cobra.Command) (string, string, ClientOptions, error) {
	hostname, err := ResolveHost(cmd)
	if err != nil {
		return "", "", ClientOptions{}, err
	}
	hosts, err := config.LoadHosts()
	if err != nil {
		return "", "", ClientOptions{}, err
	}
	e, ok := hosts.Get(hostname)
	if !ok {
		name := config.Active().CLI.Name
		return "", "", ClientOptions{}, fmt.Errorf("not authenticated to %s (run: %s auth login --hostname %s)", hostname, name, hostname)
	}
	opts := ClientOptions{Insecure: e.Insecure}
	if v, err := cmd.Root().PersistentFlags().GetBool("insecure"); err == nil && v {
		opts.Insecure = true
	}
	return hostname, e.OAuthToken, opts, nil
}
