package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/tracing"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/correlation"
)

const (
	// This header carries the JWT token for gitlab-rails
	jwtRequestHeader  = "Gitlab-Kas-Api-Request"
	jwtValidFor       = 5 * time.Second
	jwtNotBefore      = 5 * time.Second
	jwtIssuer         = "gitlab-kas"
	jwtGitLabAudience = "gitlab"

	projectIdQueryParam = "id"

	agentInfoApiPath   = "/api/v4/internal/kubernetes/agent_info"
	projectInfoApiPath = "/api/v4/internal/kubernetes/project_info"
	usagePingApiPath   = "/api/v4/internal/kubernetes/usage_metrics"
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

type gitalyInfo struct {
	Address  string            `json:"address"`
	Token    string            `json:"token"`
	Features map[string]string `json:"features"`
}

func (g *gitalyInfo) ToGitalyInfo() api.GitalyInfo {
	return api.GitalyInfo{
		Address:  g.Address,
		Token:    g.Token,
		Features: g.Features,
	}
}

type gitalyRepository struct {
	StorageName   string `json:"storage_name"`
	RelativePath  string `json:"relative_path"`
	GlRepository  string `json:"gl_repository"`
	GlProjectPath string `json:"gl_project_path"`
}

func (r *gitalyRepository) ToProtoRepository() gitalypb.Repository {
	return gitalypb.Repository{
		StorageName:   r.StorageName,
		RelativePath:  r.RelativePath,
		GlRepository:  r.GlRepository,
		GlProjectPath: r.GlProjectPath,
	}
}

type projectInfoResponse struct {
	ProjectId        int64            `json:"project_id"`
	GitalyInfo       gitalyInfo       `json:"gitaly_info"`
	GitalyRepository gitalyRepository `json:"gitaly_repository"`
}

type getAgentInfoResponse struct {
	ProjectId        int64            `json:"project_id"`
	AgentId          int64            `json:"agent_id"`
	AgentName        string           `json:"agent_name"`
	GitalyInfo       gitalyInfo       `json:"gitaly_info"`
	GitalyRepository gitalyRepository `json:"gitaly_repository"`
}

func NewClient(backend *url.URL, authSecret []byte, opts ...ClientOption) *Client {
	o := applyClientOptions(opts)
	return &Client{
		Backend: backend,
		HTTPClient: &http.Client{
			Transport: tracing.NewRoundTripper(
				correlation.NewInstrumentedRoundTripper(
					&http.Transport{
						Proxy:                 o.proxy,
						DialContext:           o.dialContext,
						MaxIdleConns:          100,
						IdleConnTimeout:       90 * time.Second,
						TLSHandshakeTimeout:   10 * time.Second,
						ResponseHeaderTimeout: 20 * time.Second,
						ExpectContinueTimeout: 1 * time.Second,
					},
					correlation.WithClientName(o.clientName),
				),
				tracing.WithRoundTripperTracer(o.tracer),
			),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		AuthSecret: authSecret,
		UserAgent:  o.userAgent,
	}
}

func (c *Client) GetAgentInfo(ctx context.Context, meta *api.AgentMeta) (*api.AgentInfo, error) {
	u := *c.Backend
	u.Path = agentInfoApiPath
	response := getAgentInfoResponse{}
	err := c.doJSON(ctx, http.MethodGet, meta, &u, nil, &response)
	if err != nil {
		return nil, err
	}
	return &api.AgentInfo{
		Meta:       *meta,
		Id:         response.AgentId,
		ProjectId:  response.ProjectId,
		Name:       response.AgentName,
		GitalyInfo: response.GitalyInfo.ToGitalyInfo(),
		Repository: response.GitalyRepository.ToProtoRepository(),
	}, nil
}

func (c *Client) GetProjectInfo(ctx context.Context, meta *api.AgentMeta, projectId string) (*api.ProjectInfo, error) {
	u := *c.Backend
	u.Path = projectInfoApiPath
	query := u.Query()
	query.Set(projectIdQueryParam, projectId)
	u.RawQuery = query.Encode()
	response := projectInfoResponse{}
	err := c.doJSON(ctx, http.MethodGet, meta, &u, nil, &response)
	if err != nil {
		return nil, err
	}
	return &api.ProjectInfo{
		ProjectId:  response.ProjectId,
		GitalyInfo: response.GitalyInfo.ToGitalyInfo(),
		Repository: response.GitalyRepository.ToProtoRepository(),
	}, nil
}

func (c *Client) SendUsage(ctx context.Context, data *UsageData) error {
	u := *c.Backend
	u.Path = usagePingApiPath
	err := c.doJSON(ctx, http.MethodPost, nil, &u, data, nil)
	if err != nil {
		return err
	}
	return nil
}

// doJSON sends a request with JSON payload (optional) and expects a response with JSON payload (optional).
// meta may be nil to avoid sending Authorization header (not acting as agentk).
// body may be nil to avoid sending a request payload.
// response may be nil to avoid sending an Accept header and ignore the response payload, if any.
func (c *Client) doJSON(ctx context.Context, method string, meta *api.AgentMeta, url *url.URL, body, response interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("json.Marshal: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	u := *url
	r, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("NewRequestWithContext: %v", err)
	}
	now := time.Now()
	claims := jwt.StandardClaims{
		Audience:  jwtGitLabAudience,
		ExpiresAt: now.Add(jwtValidFor).Unix(),
		IssuedAt:  now.Unix(),
		Issuer:    jwtIssuer,
		NotBefore: now.Add(-jwtNotBefore).Unix(),
	}
	signedClaims, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(c.AuthSecret)
	if err != nil {
		return fmt.Errorf("sign JWT: %v", err)
	}

	if meta != nil {
		r.Header.Set("Authorization", "Bearer "+string(meta.Token))
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
		return fmt.Errorf("GitLab request: %v", err)
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
	case resp.StatusCode == http.StatusForbidden: // Invalid or revoked token
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

type ErrorKind int

const (
	ErrorKindOther ErrorKind = iota
	ErrorKindForbidden
)

type ClientError struct {
	Kind       ErrorKind
	StatusCode int
}

func (c *ClientError) Error() string {
	return fmt.Sprintf("error kind: %d; status: %d", c.Kind, c.StatusCode)
}

func IsForbidden(err error) bool {
	var e *ClientError
	if !errors.As(err, &e) {
		return false
	}
	return e.Kind == ErrorKindForbidden
}
