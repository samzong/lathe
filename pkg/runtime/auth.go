package runtime

import "net/http"

type Authenticator interface {
	Apply(req *http.Request) error
}

type BearerAuth struct{ Token string }

func (a BearerAuth) Apply(req *http.Request) error {
	if a.Token != "" {
		req.Header.Set("Authorization", "Bearer "+a.Token)
	}
	return nil
}

type NoAuth struct{}

func (NoAuth) Apply(*http.Request) error { return nil }
