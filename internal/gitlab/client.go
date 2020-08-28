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
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/tracing"
)

const (
	// This header carries the JWT token for gitlab-rails
	kasRequestHeader = "Gitlab-Kas-Api-Request"
	kasJWTIssuer     = "gitlab-kas"

	projectIDQueryParam = "id"

	agentInfoApiPath   = "/api/v4/internal/kubernetes/agent_info"
	projectInfoApiPath = "/api/v4/internal/kubernetes/project_info"
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
	ProjectID        int64            `json:"project_id"`
	GitalyInfo       gitalyInfo       `json:"gitaly_info"`
	GitalyRepository gitalyRepository `json:"gitaly_repository"`
}

type getAgentInfoResponse struct {
	ProjectID        int64            `json:"project_id"`
	AgentID          int64            `json:"agent_id"`
	AgentName        string           `json:"agent_name"`
	GitalyInfo       gitalyInfo       `json:"gitaly_info"`
	GitalyRepository gitalyRepository `json:"gitaly_repository"`
}

func NewClient(backend *url.URL, authSecret []byte, userAgent string) *Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &Client{
		Backend: backend,
		HTTPClient: &http.Client{
			Transport: tracing.NewRoundTripper(
				correlation.NewInstrumentedRoundTripper(
					&http.Transport{
						Proxy:                 http.ProxyFromEnvironment,
						DialContext:           dialer.DialContext,
						MaxIdleConns:          100,
						IdleConnTimeout:       90 * time.Second,
						TLSHandshakeTimeout:   10 * time.Second,
						ResponseHeaderTimeout: 20 * time.Second,
						ExpectContinueTimeout: 1 * time.Second,
					},
				),
			),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		AuthSecret: authSecret,
		UserAgent:  userAgent,
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
		ID:         response.AgentID,
		ProjectID:  response.ProjectID,
		Name:       response.AgentName,
		GitalyInfo: response.GitalyInfo.ToGitalyInfo(),
		Repository: response.GitalyRepository.ToProtoRepository(),
	}, nil
}

func (c *Client) GetProjectInfo(ctx context.Context, meta *api.AgentMeta, projectId string) (*api.ProjectInfo, error) {
	u := *c.Backend
	u.Path = projectInfoApiPath
	query := u.Query()
	query.Set(projectIDQueryParam, projectId)
	u.RawQuery = query.Encode()
	response := projectInfoResponse{}
	err := c.doJSON(ctx, http.MethodGet, meta, &u, nil, &response)
	if err != nil {
		return nil, err
	}
	return &api.ProjectInfo{
		ProjectID:  response.ProjectID,
		GitalyInfo: response.GitalyInfo.ToGitalyInfo(),
		Repository: response.GitalyRepository.ToProtoRepository(),
	}, nil
}

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
	signedClaims, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{Issuer: "gitlab-kas"}).
		SignedString(c.AuthSecret)
	if err != nil {
		return fmt.Errorf("sign JWT: %v", err)
	}

	r.Header.Set("Authorization", "Bearer "+string(meta.Token))
	r.Header.Set(kasRequestHeader, signedClaims)
	r.Header.Set("User-Agent", c.UserAgent)
	r.Header.Set("Accept", "application/json")
	if bodyReader != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return fmt.Errorf("GitLab request: %v", err)
	}
	defer resp.Body.Close() // nolint: errcheck
	switch resp.StatusCode {
	case http.StatusOK: // Handled below
	case http.StatusForbidden: // Invalid or revoked token
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