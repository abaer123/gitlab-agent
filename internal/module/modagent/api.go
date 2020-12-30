package modagent

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/cli-runtime/pkg/resource"
)

// Config holds configuration for a Module.
type Config struct {
	Log       *zap.Logger
	AgentMeta *modshared.AgentMeta
	Api       API
	// K8sClientGetter provides means to interact with the Kubernetes cluster agentk is running in.
	K8sClientGetter resource.RESTClientGetter
	// KasConn is the gRPC connection to gitlab-kas.
	KasConn grpc.ClientConnInterface
}

type GitLabResponse struct {
	Status     string // e.g. "200 OK"
	StatusCode int32  // e.g. 200
	Header     http.Header
	Body       io.ReadCloser
}

// API provides the API for the module to use.
type API interface {
	MakeGitLabRequest(ctx context.Context, path string, opts ...GitLabRequestOption) (*GitLabResponse, error)
}

type Factory interface {
	// New creates a new instance of a Module.
	New(*Config) (Module, error)
	// Name returns module's name.
	Name() string
}

type Module interface {
	// Run starts the module.
	// Run can block until the context is canceled or exit with nil if there is nothing to do.
	// cfg is a channel that gets configuration updates sent to it. It's closed when the module should shut down.
	// cfg is a shared instance, must not be mutated. Module should make a copy if it needs to mutate the object.
	// Applying configuration may take time, the provided context may signal done if module should shut down.
	// cfg only provides the latest available configuration, intermediate configuration states are discarded.
	Run(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) error
	// DefaultAndValidateConfiguration applies defaults and validates the passed configuration.
	// It is called each time on configuration update before sending it via the channel passed to Run().
	// cfg is a shared instance, module can mutate only the part of it that it owns and only inside of this method.
	DefaultAndValidateConfiguration(cfg *agentcfg.AgentConfiguration) error
	// Name returns module's name.
	Name() string
}

type GitLabRequestConfig struct {
	Method  string
	Headers http.Header
	Query   url.Values
	Body    io.ReadCloser
}

func defaultRequestConfig() *GitLabRequestConfig {
	return &GitLabRequestConfig{
		Method:  http.MethodGet,
		Headers: make(http.Header),
		Query:   make(url.Values),
	}
}

func ApplyRequestOptions(opts []GitLabRequestOption) *GitLabRequestConfig {
	c := defaultRequestConfig()
	for _, o := range opts {
		o(c)
	}
	return c
}

type GitLabRequestOption func(*GitLabRequestConfig)

func WithRequestHeaders(headers http.Header) GitLabRequestOption {
	return func(c *GitLabRequestConfig) {
		c.Headers = headers
	}
}

func WithRequestHeader(header string, values ...string) GitLabRequestOption {
	return func(c *GitLabRequestConfig) {
		c.Headers[header] = values
	}
}

func WithRequestQueryParam(key string, values ...string) GitLabRequestOption {
	return func(c *GitLabRequestConfig) {
		c.Query[key] = values
	}
}

func WithRequestQuery(query url.Values) GitLabRequestOption {
	return func(c *GitLabRequestConfig) {
		c.Query = query
	}
}

// WithRequestBody specifies request body to send.
// If body implements io.ReadCloser, its Close() method will be called once the data has been sent.
func WithRequestBody(body io.Reader) GitLabRequestOption {
	return func(c *GitLabRequestConfig) {
		if rc, ok := body.(io.ReadCloser); ok {
			c.Body = rc
		} else {
			c.Body = ioutil.NopCloser(body)
		}
	}
}
