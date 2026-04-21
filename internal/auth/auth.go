package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/samzong/lathe/internal/config"
	"github.com/samzong/lathe/internal/runtime"
)

func NewCommand(m *config.Manifest) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: fmt.Sprintf("Authenticate %s with a host", m.CLI.Name),
	}
	cmd.AddCommand(newLogin(m), newStatus(), newLogout())
	return cmd
}

func rootString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Root().PersistentFlags().GetString(name)
	return v
}

func rootBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool(name)
	return v
}

func newLogin(m *config.Manifest) *cobra.Command {
	var (
		withToken    bool
		skipValidate bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a host",
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := rootString(cmd, "hostname")
			insecure := rootBool(cmd, "insecure")
			if hostname == "" && !withToken {
				fmt.Fprint(os.Stderr, "? Hostname: ")
				line, err := readLine(os.Stdin)
				if err != nil {
					return err
				}
				hostname = strings.TrimSpace(line)
			}
			if hostname == "" {
				return errors.New("hostname is required (use --hostname)")
			}
			hostname = config.NormalizeHostname(hostname)

			token, err := readToken(withToken)
			if err != nil {
				return err
			}
			if token == "" {
				return errors.New("empty token")
			}

			user := ""
			if !skipValidate {
				result, err := validateToken(cmd.Context(), hostname, token, m.Auth.Validate, runtime.ClientOptions{Insecure: insecure})
				if err != nil {
					if !insecure && strings.Contains(err.Error(), "tls:") {
						return fmt.Errorf("token validation failed against %s: %w\n\nThe server uses a self-signed or non-standard certificate. Re-run with --insecure to skip TLS verification (the choice is persisted per host).", hostname, err)
					}
					return fmt.Errorf("token validation failed against %s: %w", hostname, err)
				}
				user = result.Username
				if user != "" {
					fmt.Fprintf(os.Stderr, "✓ Authenticated as %s\n", user)
				}
			}

			hosts, err := config.LoadHosts()
			if err != nil {
				return err
			}
			hosts.Set(hostname, config.HostEntry{User: user, OAuthToken: token, Insecure: insecure})
			if err := hosts.Save(); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "✓ Logged in to %s\n", hostname)
			return nil
		},
	}
	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read token from stdin")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Do not validate the token against the server")
	return cmd
}

func newStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "View authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := rootString(cmd, "hostname")
			hosts, err := config.LoadHosts()
			if err != nil {
				return err
			}
			names := hosts.Names()
			if len(names) == 0 {
				return runtime.NewNotAuthenticatedError()
			}
			if hostname != "" {
				e, ok := hosts.Get(hostname)
				if !ok {
					return fmt.Errorf("not logged in to %s", hostname)
				}
				printStatus(hostname, e)
				return nil
			}
			for _, n := range names {
				e, _ := hosts.Get(n)
				printStatus(n, e)
			}
			return nil
		},
	}
}

func newLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove authentication for a host",
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := rootString(cmd, "hostname")
			hosts, err := config.LoadHosts()
			if err != nil {
				return err
			}
			names := hosts.Names()
			if len(names) == 0 {
				return errors.New("not logged in to any host")
			}
			if hostname == "" {
				if len(names) == 1 {
					hostname = names[0]
				} else {
					return fmt.Errorf("multiple hosts configured, specify --hostname (have: %s)", strings.Join(names, ", "))
				}
			}
			if !hosts.Delete(hostname) {
				return fmt.Errorf("not logged in to %s", hostname)
			}
			if err := hosts.Save(); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "✓ Logged out of %s\n", hostname)
			return nil
		},
	}
}

func printStatus(hostname string, e config.HostEntry) {
	user := e.User
	if user == "" {
		user = fmt.Sprintf("(unknown — run `%s auth login` to validate)", config.Active().CLI.Name)
	}
	fmt.Fprintf(os.Stdout, "%s\n  ✓ Logged in as %s\n  ✓ Token: %s\n", hostname, user, maskToken(e.OAuthToken))
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return "****"
	}
	return "****" + t[len(t)-4:]
}

func readToken(fromStdin bool) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "? Paste your authentication token: ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := readLine(os.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return line, nil
}
