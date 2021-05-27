package gitlab

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
)

type ResponseHandler interface {
	// Handle is invoked with HTTP client's response and error values.
	Handle(*http.Response, error) error
	// Accept returns the value to send in the Accept HTTP header.
	// Empty string means no value should be sent.
	Accept() string
}

// doConfig holds configuration for the Do call.
type doConfig struct {
	method          string
	path            string
	query           url.Values
	header          http.Header
	body            io.Reader
	responseHandler ResponseHandler
	withJWT         bool
}

func (c *doConfig) ensureHeaderNotNil() {
	if c.header == nil {
		c.header = make(http.Header)
	}
}

// DoOption to configure the Do call of the client.
type DoOption func(*doConfig) error

func applyDoOptions(opts []DoOption) (doConfig, error) {
	config := doConfig{
		method: http.MethodGet,
		path:   "/",
	}
	for _, v := range opts {
		err := v(&config)
		if err != nil {
			return doConfig{}, err
		}
	}
	if config.responseHandler == nil {
		return doConfig{}, errors.New("missing response handler")
	}

	return config, nil
}

func WithMethod(method string) DoOption {
	return func(config *doConfig) error {
		config.method = method
		return nil
	}
}

func WithPath(path string) DoOption {
	return func(config *doConfig) error {
		config.path = path
		return nil
	}
}

func WithQuery(query url.Values) DoOption {
	return func(config *doConfig) error {
		config.query = query
		return nil
	}
}

func WithHeader(header http.Header) DoOption {
	return func(config *doConfig) error {
		clone := header.Clone()
		if config.header == nil {
			config.header = clone
		} else {
			for k, v := range clone {
				config.header[k] = v // overwrite
			}
		}
		return nil
	}
}

func WithJWT(withJWT bool) DoOption {
	return func(config *doConfig) error {
		config.withJWT = withJWT
		return nil
	}
}

func WithAgentToken(agentToken api.AgentToken) DoOption {
	return func(config *doConfig) error {
		config.ensureHeaderNotNil()
		config.header.Set("Authorization", "Bearer "+string(agentToken))
		return nil
	}
}

func WithJobToken(jobToken string) DoOption {
	return func(config *doConfig) error {
		config.ensureHeaderNotNil()
		config.header.Set("Job-Token", jobToken)
		return nil
	}
}

// WithRequestBody sets the request body and HTTP Content-Type header if contentType is not empty.
func WithRequestBody(body io.Reader, contentType string) DoOption {
	return func(config *doConfig) error {
		config.body = body
		if contentType != "" {
			config.ensureHeaderNotNil()
			config.header.Set("Content-Type", contentType)
		}
		return nil
	}
}

func WithJsonRequestBody(body interface{}) DoOption {
	return func(config *doConfig) error {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("WithJsonRequestBody: json.Marshal: %v", err)
		}
		return WithRequestBody(bytes.NewReader(bodyBytes), "application/json")(config)
	}
}

func WithResponseHandler(handler ResponseHandler) DoOption {
	return func(config *doConfig) error {
		config.responseHandler = handler
		accept := handler.Accept()
		if accept != "" {
			config.ensureHeaderNotNil()
			config.header.Set("Accept", accept)
		}
		return nil
	}
}
