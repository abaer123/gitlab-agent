package agentkapp

import (
	"context"
	"errors"
	"fmt"
	"io"

	gitlab_access_rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	maxDataChunkSize = 32 * 1024

	headerFieldNumber  protoreflect.FieldNumber = 1
	dataFieldNumber    protoreflect.FieldNumber = 2
	trailerFieldNumber protoreflect.FieldNumber = 3
)

// agentAPI is an implementation of modagent.API.
type agentAPI struct {
	ModuleName      string
	Client          gitlab_access_rpc.GitlabAccessClient
	ResponseVisitor *grpctool.StreamVisitor
}

// Capture does nothing at the moment
func (a *agentAPI) Capture(err error, opts ...errortracking.CaptureOption) {
}

func (a *agentAPI) MakeGitLabRequest(ctx context.Context, path string, opts ...modagent.GitLabRequestOption) (*modagent.GitLabResponse, error) {
	config := modagent.ApplyRequestOptions(opts)
	ctx, cancel := context.WithCancel(ctx)
	client, errReq := a.Client.MakeRequest(ctx)
	if errReq != nil {
		cancel()
		if config.Body != nil {
			_ = config.Body.Close()
		}
		return nil, errReq
	}
	response := make(chan *modagent.GitLabResponse)
	responseErr := make(chan error)
	pr, pw := io.Pipe()

	// Write request
	go func() {
		err := a.makeRequest(client, path, config)
		if err != nil {
			cancel()
			_ = pw.CloseWithError(err)
		}
	}()
	// Read response
	go func() {
		responseSent := false
		err := a.ResponseVisitor.Visit(client,
			grpctool.WithCallback(headerFieldNumber, func(header *gitlab_access_rpc.Response_Header) error {
				resp := &modagent.GitLabResponse{
					Status:     header.Response.Status,
					StatusCode: header.Response.StatusCode,
					Header:     header.Response.HttpHeader(),
					Body: cancelingReadCloser{
						ReadCloser: pr,
						cancel:     cancel,
					},
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case response <- resp:
					responseSent = true
					return nil
				}
			}),
			grpctool.WithCallback(dataFieldNumber, func(data *gitlab_access_rpc.Response_Data) error {
				_, err := pw.Write(data.Data)
				return err
			}),
			grpctool.WithCallback(trailerFieldNumber, func(trailer *gitlab_access_rpc.Response_Trailer) error {
				return nil
			}),
			grpctool.WithEOFCallback(func() error {
				return pw.Close()
			}),
		)
		if err != nil {
			cancel()
			// This aborts the reader if the stream has not been closed already. Otherwise a no-op.
			_ = pw.CloseWithError(err)
			if !responseSent {
				// If we are here, there was an error before we've received headers.
				select {
				case <-ctx.Done():
				case responseErr <- err:
				}
			}
		}
	}()
	select {
	case <-ctx.Done():
		err := ctx.Err()
		_ = pw.CloseWithError(err)
		return nil, err
	case resp := <-response:
		return resp, nil
	case err := <-responseErr:
		return nil, err
	}
}

func (a *agentAPI) makeRequest(client gitlab_access_rpc.GitlabAccess_MakeRequestClient, path string, config *modagent.GitLabRequestConfig) (retErr error) {
	defer errz.SafeClose(config.Body, &retErr)
	err := client.Send(&gitlab_access_rpc.Request{
		Message: &gitlab_access_rpc.Request_Header_{
			Header: &gitlab_access_rpc.Request_Header{
				ModuleName: a.ModuleName,
				Request: &prototool.HttpRequest{
					Method:  config.Method,
					Header:  prototool.HttpHeaderToValuesMap(config.Header),
					UrlPath: path,
					Query:   prototool.UrlValuesToValuesMap(config.Query),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send request header: %w", err) // wrap
	}
	if config.Body != nil {
		buffer := make([]byte, maxDataChunkSize)
		for {
			var n int
			n, err = config.Body.Read(buffer)
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("send request body: %w", err) // wrap
			}
			if n > 0 { // handle n=0, err=io.EOF case
				sendErr := client.Send(&gitlab_access_rpc.Request{
					Message: &gitlab_access_rpc.Request_Data_{
						Data: &gitlab_access_rpc.Request_Data{
							Data: buffer[:n],
						}},
				})
				if sendErr != nil {
					return fmt.Errorf("send request data: %w", sendErr) // wrap
				}
			}
			if errors.Is(err, io.EOF) {
				break
			}
		}
	}
	err = client.Send(&gitlab_access_rpc.Request{
		Message: &gitlab_access_rpc.Request_Trailer_{
			Trailer: &gitlab_access_rpc.Request_Trailer{},
		},
	})
	if err != nil {
		return fmt.Errorf("send request trailer: %w", err) // wrap
	}
	err = client.CloseSend()
	if err != nil {
		return fmt.Errorf("close request stream: %w", err) // wrap
	}
	return nil
}

type cancelingReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c cancelingReadCloser) Close() error {
	c.cancel()
	return c.ReadCloser.Close()
}
