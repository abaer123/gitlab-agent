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

type fetchAgentInfoRequest struct {
	AgentVersion string `json:"agent_version"`
}

type fetchAgentInfoResponse struct {
	StorageName   string `json:"storage_name"`
	RelativePath  string `json:"relative_path"`
	GlRepository  string `json:"gl_repository"`
	GlProjectPath string `json:"gl_project_path"`
}

func NewClient(backend *url.URL, socket string) *Client {
	return &Client{
		Backend: backend,
		HttpClient: &http.Client{
			Transport: roundtripper.NewBackendRoundTripper(backend, socket, responseHeaderTimeout),
		},
	}
}

func (c *Client) FetchAgentInfo(ctx context.Context, meta *api.AgentMeta) (*api.AgentInfo, error) {
	body, err := json.Marshal(fetchAgentInfoRequest{
		AgentVersion: meta.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %v", err)
	}
	u := *c.Backend
	u.Path = "/api/v4/internal/kubernetes/allowed"
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %v", err)
	}
	r.Header.Set("Authorization", "Bearer "+string(meta.Token))
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.HttpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("GitLab request: %v", err)
	}
	defer drainAndClose(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK: // Handled below
	case http.StatusForbidden: // Invalid or revoked token
		return nil, &ClientError{
			Kind:       ErrorKindForbidden,
			StatusCode: http.StatusForbidden,
		}
	default: // Unexpected status
		return nil, &ClientError{
			Kind:       ErrorKindOther,
			StatusCode: resp.StatusCode,
		}
	}
	if !isApplicationJson(resp) {
		return nil, fmt.Errorf("unexpected Content-Type in response: %q", r.Header.Get("Content-Type"))
	}
	response := fetchAgentInfoResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("json.Decode: %v", err)
	}
	return &api.AgentInfo{
		Meta: *meta,
		Name: "agent-ash2k", // TODO fetch from GitLab
		Repository: api.AgentConfigRepository{
			StorageName:   response.StorageName,
			RelativePath:  response.RelativePath + ".git",
			GlRepository:  response.GlRepository,
			GlProjectPath: response.GlProjectPath,
		},
	}, nil
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
