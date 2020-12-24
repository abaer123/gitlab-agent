package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	headersFieldNumber  protoreflect.FieldNumber = 1
	dataFieldNumber     protoreflect.FieldNumber = 2
	trailersFieldNumber protoreflect.FieldNumber = 3

	urlPathForModules = "/api/v4/internal/kubernetes/modules/"
	maxDataChunkSize  = 32 * 1024
)

type module struct {
	api           modserver.API
	gitLabClient  gitlab.ClientInterface
	streamVisitor *grpctool.StreamVisitor
}

func (m *module) MakeRequest(server rpc.GitlabAccess_MakeRequestServer) error {
	ctx := server.Context()
	agentToken := api.AgentTokenFromContext(ctx)
	log := grpctool.LoggerFromContext(ctx)

	g, ctx := errgroup.WithContext(ctx) // if one of the goroutines returns a non-nil error, ctx is canceled.

	pr, pw := io.Pipe()

	// A channel to pass received Headers message to the other goroutine so that it can make an HTTP call.
	headersMsg := make(chan *rpc.Request_Headers)

	// Pipe gRPC request -> HTTP request
	g.Go(func() error {
		return m.streamVisitor.Visit(server,
			grpctool.WithCallback(headersFieldNumber, func(headers *rpc.Request_Headers) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case headersMsg <- headers:
					return nil
				}
			}),
			grpctool.WithCallback(dataFieldNumber, func(data *rpc.Request_Data) error {
				_, err := pw.Write(data.Data)
				return err
			}),
			grpctool.WithCallback(trailersFieldNumber, func(trailers *rpc.Request_Trailers) error {
				// Nothing to do
				return nil
			}),
			grpctool.WithEOFCallback(func() error {
				return pw.Close() // Nothing more to send, close the write end of the pipe
			}),
		)
	})
	// Pipe HTTP response -> gRPC response
	g.Go(func() (retErr error) {
		// Make sure the writer is unblocked if we exit abruptly
		// The error is ignored because it will always occur if things go normally - the pipe will have been
		// closed already when this code is reached (and that's an error).
		defer pr.Close() // nolint: errcheck
		var h *rpc.Request_Headers
		select {
		case <-ctx.Done():
			return ctx.Err()
		case h = <-headersMsg:
		}
		urlPath := urlPathForModules + url.PathEscape(h.ModuleName) + h.UrlPath
		resp, err := m.gitLabClient.DoStream(ctx, h.Method, urlPath, h.ToHttpHeader(), h.ToUrlQuery(), agentToken, pr) // nolint:bodyclose
		if err != nil {
			return err
		}
		defer errz.SafeClose(resp.Body, &retErr)

		err = server.Send(&rpc.Response{
			Message: &rpc.Response_Headers_{
				Headers: &rpc.Response_Headers{
					StatusCode: int32(resp.StatusCode),
					Status:     resp.Status,
					Headers:    rpc.HeadersFromHttpHeaders(resp.Header),
				}},
		})
		if err != nil {
			return m.api.HandleSendError(log, "MakeRequest failed to send headers", err)
		}

		buffer := make([]byte, maxDataChunkSize)
		for {
			var n int
			n, err = resp.Body.Read(buffer)
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("read response body: %w", err) // wrap
			}
			if n > 0 { // handle n=0, err=io.EOF case
				sendErr := server.Send(&rpc.Response{
					Message: &rpc.Response_Data_{
						Data: &rpc.Response_Data{
							Data: buffer[:n],
						}},
				})
				if sendErr != nil {
					return m.api.HandleSendError(log, "MakeRequest failed to send data", sendErr)
				}
			}
			if errors.Is(err, io.EOF) {
				break
			}
		}
		err = server.Send(&rpc.Response{
			Message: &rpc.Response_Trailers_{
				Trailers: &rpc.Response_Trailers{},
			},
		})
		if err != nil {
			return m.api.HandleSendError(log, "MakeRequest failed to send trailers", err)
		}
		return nil
	})
	err := g.Wait()
	if err != nil {
		switch {
		case errz.ContextDone(err):
			err = status.Error(codes.Unavailable, "unavailable")
		case status.Code(err) != codes.Unknown:
			// A gRPC status already
		default:
			m.api.LogAndCapture(ctx, log, "MakeRequest()", err)
			err = status.Error(codes.Unavailable, "unavailable")
		}
	}
	return err
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) Name() string {
	return gitlab_access.ModuleName
}
