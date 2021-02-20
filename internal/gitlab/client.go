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
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
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
// agentToken may be empty to avoid sending Authorization header (not acting as agentk).
// body may be nil to avoid sending a request payload.
// response may be nil to avoid sending an Accept header and ignore the response payload, if any.
// query may be nil to avoid sending any URL query parameters.
func (c *Client) DoJSON(ctx context.Context, method, path string, query url.Values, agentToken api.AgentToken, body, response interface{}) (retErr error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("json.Marshal: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	h := make(http.Header)
	if response != nil {
		h.Set("Accept", "application/json")
	}
	if bodyReader != nil {
		h.Set("Content-Type", "application/json")
	}
	resp, err := c.DoStream(ctx, method, path, h, query, agentToken, bodyReader)
	if err != nil {
		return err
	}
	defer errz.SafeClose(resp.Body, &retErr)
	switch {
	case resp.StatusCode == http.StatusOK:
		if response == nil {
			// No response expected or ignoring response
			return nil
		}
		if !isApplicationJSON(resp) {
			return fmt.Errorf("unexpected Content-Type in response: %q", resp.Header.Get("Content-Type"))
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("request body read: %v", err)
		}
		if err = json.Unmarshal(data, response); err != nil {
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

// DoStream can be used to access GitLab API using streams rather than fixed size payloads.
// agentToken may be empty to avoid sending Authorization header (not acting as agentk).
// body may be nil to avoid sending a request payload.
// query may be nil to avoid sending any URL query parameters.
// header may be used to send extra HTTP header. May be nil.
func (c *Client) DoStream(ctx context.Context, method, path string, header http.Header, query url.Values, agentToken api.AgentToken, body io.Reader) (*http.Response, error) {
	u := *c.Backend
	u.Path = path
	u.RawQuery = query.Encode() // handles query == nil
	r, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %v", err)
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
		return nil, fmt.Errorf("sign JWT: %v", err)
	}
	if header != nil {
		r.Header = header.Clone()
	}
	if agentToken != "" {
		r.Header.Set("Authorization", "Bearer "+string(agentToken))
	}
	r.Header.Set(jwtRequestHeader, signedClaims)
	if c.UserAgent != "" {
		r.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		select {
		case <-ctx.Done(): // assume request errored out because of context
			return nil, ctx.Err()
		default:
			return nil, fmt.Errorf("GitLab request: %v", err)
		}
	}
	return resp, nil
}

func isContentType(expected, actual string) bool {
	parsed, _, err := mime.ParseMediaType(actual)
	return err == nil && parsed == expected
}

func isApplicationJSON(r *http.Response) bool {
	contentType := r.Header.Get("Content-Type")
	return isContentType("application/json", contentType)
}
