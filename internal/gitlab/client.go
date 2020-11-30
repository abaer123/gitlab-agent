package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/httpz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/tracing"
	"gitlab.com/gitlab-org/labkit/correlation"
)

const (
	// This header carries the JWT token for gitlab-rails
	jwtRequestHeader  = "Gitlab-Kas-Api-Request"
	jwtValidFor       = 5 * time.Second
	jwtNotBefore      = 5 * time.Second
	jwtIssuer         = "gitlab-kas"
	jwtGitLabAudience = "gitlab"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	Backend    *url.URL
	HTTPClient HTTPClient
	AuthSecret []byte
	UserAgent  string
}

func NewClient(backend *url.URL, authSecret []byte, opts ...ClientOption) *Client {
	o := applyClientOptions(opts)
	var transport http.RoundTripper = &http.Transport{
		Proxy:                 o.proxy,
		DialContext:           o.dialContext,
		TLSClientConfig:       o.tlsConfig,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if o.limiter != nil {
		transport = &httpz.RateLimitingRoundTripper{
			Delegate: transport,
			Limiter:  o.limiter,
		}
	}
	return &Client{
		Backend: backend,
		HTTPClient: &http.Client{
			Transport: tracing.NewRoundTripper(
				correlation.NewInstrumentedRoundTripper(
					transport,
					correlation.WithClientName(o.clientName),
				),
				tracing.WithRoundTripperTracer(o.tracer),
				tracing.WithLogger(o.log),
			),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		AuthSecret: authSecret,
		UserAgent:  o.userAgent,
	}
}

// DoJSON sends a request with JSON payload (optional) and expects a response with JSON payload (optional).
// meta may be nil to avoid sending Authorization header (not acting as agentk).
// body may be nil to avoid sending a request payload.
// response may be nil to avoid sending an Accept header and ignore the response payload, if any.
func (c *Client) DoJSON(ctx context.Context, method, path string, query url.Values, agentMeta *api.AgentMeta, body, response interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("json.Marshal: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	u := *c.Backend
	u.Path = path
	u.RawQuery = query.Encode() // handles query == nil
	r, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("NewRequestWithContext: %v", err)
	}
	now := time.Now()
	claims := jwt.StandardClaims{
		Audience:  jwt.ClaimStrings{jwtGitLabAudience},
		ExpiresAt: jwt.At(now.Add(jwtValidFor)),
		IssuedAt:  jwt.At(now),
		Issuer:    jwtIssuer,
		NotBefore: jwt.At(now.Add(-jwtNotBefore)),
	}
	signedClaims, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(c.AuthSecret)
	if err != nil {
		return fmt.Errorf("sign JWT: %v", err)
	}

	if agentMeta != nil {
		r.Header.Set("Authorization", "Bearer "+string(agentMeta.Token))
	}
	r.Header.Set(jwtRequestHeader, signedClaims)
	if c.UserAgent != "" {
		r.Header.Set("User-Agent", c.UserAgent)
	}
	if response != nil {
		r.Header.Set("Accept", "application/json")
	}
	if bodyReader != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		select {
		case <-ctx.Done(): // assume request errored out because of context
			return ctx.Err()
		default:
			return fmt.Errorf("GitLab request: %v", err)
		}
	}
	defer resp.Body.Close() // nolint: errcheck
	switch {
	case resp.StatusCode == http.StatusOK:
		if response == nil {
			// No response expected or ignoring response
			return nil
		}
		if !isApplicationJSON(resp) {
			return fmt.Errorf("unexpected Content-Type in response: %q", r.Header.Get("Content-Type"))
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("request body read: %v", err)
		}
		if err := json.Unmarshal(data, response); err != nil {
			return fmt.Errorf("json.Unmarshal: %v", err)
		}
		return nil
	case resp.StatusCode == http.StatusNoContent && response == nil:
		return nil
	case resp.StatusCode == http.StatusUnauthorized: // No token, invalid token, revoked token
		return &ClientError{
			Kind:       ErrorKindUnauthorized,
			StatusCode: http.StatusUnauthorized,
		}
	case resp.StatusCode == http.StatusForbidden: // Access denied
		return &ClientError{
			Kind:       ErrorKindForbidden,
			StatusCode: http.StatusForbidden,
		}
	default: // Unexpected status
		return &ClientError{
			Kind:       ErrorKindOther,
			StatusCode: resp.StatusCode,
		}
	}
}

func isContentType(expected, actual string) bool {
	parsed, _, err := mime.ParseMediaType(actual)
	return err == nil && parsed == expected
}

func isApplicationJSON(r *http.Response) bool {
	contentType := r.Header.Get("Content-Type")
	return isContentType("application/json", contentType)
}
