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

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/gitlab/fromworkhorse/roundtripper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	responseHeaderTimeout = 20 * time.Second
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	Backend    *url.URL
	HttpClient HttpClient
}

type manifestProjectInfoResponse struct {
	ProjectId int64 `json:"project_id"`
	gitalyRepositoryResponsePart
}

type gitalyRepositoryResponsePart struct {
	StorageName   string `json:"storage_name"`
	RelativePath  string `json:"relative_path"`
	GlRepository  string `json:"gl_repository"`
	GlProjectPath string `json:"gl_project_path"`
}

type getAgentInfoResponse struct {
	ProjectId int64  `json:"project_id"`
	AgentId   int64  `json:"agent_id"`
	AgentName string `json:"agent_name"`
	gitalyRepositoryResponsePart
}

func NewClient(backend *url.URL, socket string) *Client {
	return &Client{
		Backend: backend,
		HttpClient: &http.Client{
			Transport: roundtripper.NewBackendRoundTripper(backend, socket, responseHeaderTimeout),
		},
	}
}

func (c *Client) GetAgentInfo(ctx context.Context, meta *api.AgentMeta) (*api.AgentInfo, error) {
	u := *c.Backend
	u.Path = "/api/v4/internal/kubernetes/agent_info"
	response := getAgentInfoResponse{}
	err := c.doJSON(ctx, http.MethodGet, meta, &u, nil, &response)
	if err != nil {
		return nil, err
	}
	return &api.AgentInfo{
		Meta: *meta,
		Id:   response.AgentId,
		Name: response.AgentName,
		Repository: gitalypb.Repository{
			StorageName:   response.StorageName,
			RelativePath:  response.RelativePath,
			GlRepository:  response.GlRepository,
			GlProjectPath: response.GlProjectPath,
		},
	}, nil
}

func (c *Client) GetProjectInfo(ctx context.Context, meta *api.AgentMeta, projectId string) (*api.ProjectInfo, error) {
	u := *c.Backend
	u.Path = "/api/v4/internal/kubernetes/project_info"
	query := u.Query()
	query.Set("id", projectId)
	u.RawQuery = query.Encode()
	response := manifestProjectInfoResponse{}
	err := c.doJSON(ctx, http.MethodGet, meta, &u, nil, &response)
	if err != nil {
		return nil, err
	}
	return &api.ProjectInfo{
		ProjectId: response.ProjectId,
		Repository: gitalypb.Repository{
			StorageName:   response.StorageName,
			RelativePath:  response.RelativePath,
			GlRepository:  response.GlRepository,
			GlProjectPath: response.GlProjectPath,
		},
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
	query := u.Query()
	query.Set("agentk_version", meta.Version)
	u.RawQuery = query.Encode()
	r, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("NewRequestWithContext: %v", err)
	}
	r.Header.Set("Authorization", "Bearer "+string(meta.Token))
	r.Header.Set("User-Agent", "kas")
	r.Header.Set("Accept", "application/json")
	if bodyReader != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HttpClient.Do(r)
	if err != nil {
		return fmt.Errorf("GitLab request: %v", err)
	}
	defer drainAndClose(resp.Body)
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
	if !isApplicationJson(resp) {
		return fmt.Errorf("unexpected Content-Type in response: %q", r.Header.Get("Content-Type"))
	}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("json.Decode: %v", err)
	}
	return nil
}

func isContentType(expected, actual string) bool {
	parsed, _, err := mime.ParseMediaType(actual)
	return err == nil && parsed == expected
}

func isApplicationJson(r *http.Response) bool {
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

func drainAndClose(body io.ReadCloser) {
	defer body.Close()
	io.Copy(ioutil.Discard, io.LimitReader(body, 16*1024))
}
