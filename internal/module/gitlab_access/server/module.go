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
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	headerFieldNumber  protoreflect.FieldNumber = 1
	dataFieldNumber    protoreflect.FieldNumber = 2
	trailerFieldNumber protoreflect.FieldNumber = 3

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

	// A channel to pass received Header message to the other goroutine so that it can make an HTTP call.
	headerMsg := make(chan *rpc.Request_Header)

	// Pipe gRPC request -> HTTP request
	g.Go(func() error {
		return m.streamVisitor.Visit(server,
			grpctool.WithCallback(headerFieldNumber, func(header *rpc.Request_Header) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case headerMsg <- header:
					return nil
				}
			}),
			grpctool.WithCallback(dataFieldNumber, func(data *rpc.Request_Data) error {
				_, err := pw.Write(data.Data)
				return err
			}),
			grpctool.WithCallback(trailerFieldNumber, func(trailer *rpc.Request_Trailer) error {
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
		var h *rpc.Request_Header
		select {
		case <-ctx.Done():
			return ctx.Err()
		case h = <-headerMsg:
		}
		urlPath := urlPathForModules + url.PathEscape(h.ModuleName) + h.Request.UrlPath
		resp, err := m.gitLabClient.DoStream(ctx, h.Request.Method, urlPath, h.Request.HttpHeader(), h.Request.UrlQuery(), agentToken, pr) // nolint:bodyclose
		if err != nil {
			return err
		}
		defer errz.SafeClose(resp.Body, &retErr)

		err = server.Send(&rpc.Response{
			Message: &rpc.Response_Header_{
				Header: &rpc.Response_Header{
					Response: &prototool.HttpResponse{
						StatusCode: int32(resp.StatusCode),
						Status:     resp.Status,
						Header:     prototool.ValuesMapFromHttpHeader(resp.Header),
					},
				},
			},
		})
		if err != nil {
			return m.api.HandleSendError(log, "MakeRequest failed to send header", err)
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
			Message: &rpc.Response_Trailer_{
				Trailer: &rpc.Response_Trailer{},
			},
		})
		if err != nil {
			return m.api.HandleSendError(log, "MakeRequest failed to send trailer", err)
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
			m.api.HandleProcessingError(ctx, log, "MakeRequest()", err)
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
