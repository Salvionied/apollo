package backendutil

import (
	"errors"
	"net/http"
)

// SameOriginRedirect confines provider requests to the configured origin.
func SameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host {
		return errors.New("provider redirect crosses its configured origin")
	}
	return nil
}
