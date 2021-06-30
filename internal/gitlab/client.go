package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/httpz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/tracing"
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

func (c *Client) Do(ctx context.Context, opts ...DoOption) error {
	o, err := applyDoOptions(opts)
	if err != nil {
		return err
	}
	u := *c.Backend
	u.Path = o.path
	u.RawQuery = o.query.Encode() // handles query == nil
	r, err := http.NewRequestWithContext(ctx, o.method, u.String(), o.body)
	if err != nil {
		return fmt.Errorf("NewRequestWithContext: %w", err)
	}
	if o.header != nil {
		r.Header = o.header
	}
	if o.withJWT {
		now := time.Now()
		claims := jwt.StandardClaims{
			Audience:  jwt.ClaimStrings{jwtGitLabAudience},
			ExpiresAt: jwt.At(now.Add(jwtValidFor)),
			IssuedAt:  jwt.At(now),
			Issuer:    jwtIssuer,
			NotBefore: jwt.At(now.Add(-jwtNotBefore)),
		}
		signedClaims, claimsErr := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
			SignedString(c.AuthSecret)
		if claimsErr != nil {
			return fmt.Errorf("sign JWT: %w", claimsErr)
		}
		r.Header.Set(jwtRequestHeader, signedClaims)
	}
	if c.UserAgent != "" {
		r.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.HTTPClient.Do(r) // nolint: bodyclose
	if err != nil {
		select {
		case <-ctx.Done(): // assume request errored out because of context
			err = ctx.Err()
		default:
		}
	}
	return o.responseHandler.Handle(resp, err)
}
