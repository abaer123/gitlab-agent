package agentkapp

import (
	"context"
	"errors"
	"fmt"
	"io"

	gitlab_access_rpc "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	maxDataChunkSize = 32 * 1024

	headerFieldNumber  protoreflect.FieldNumber = 1
	dataFieldNumber    protoreflect.FieldNumber = 2
	trailerFieldNumber protoreflect.FieldNumber = 3
)

// agentAPI is an implementation of modagent.API.
type agentAPI struct {
	moduleName      string
	client          gitlab_access_rpc.GitlabAccessClient
	responseVisitor *grpctool.StreamVisitor
	featureTracker  *featureTracker
}

func (a *agentAPI) HandleProcessingError(ctx context.Context, log *zap.Logger, msg string, err error) {
	if grpctool.RequestCanceled(err) {
		// An error caused by context signalling done
		return
	}
	var ue errz.UserError
	isUserError := errors.As(err, &ue)
	if isUserError {
		// TODO Don't log it, send it somewhere the user can see it https://gitlab.com/gitlab-org/gitlab/-/issues/277323
		// Log at Info for now.
		log.Info(msg, zap.Error(err))
	} else {
		a.logAndCapture(ctx, log, msg, err)
	}
}

func (a *agentAPI) HandleSendError(log *zap.Logger, msg string, err error) error {
	// The problem is almost certainly with the client's connection.
	// Still log it on Debug.
	if !grpctool.RequestCanceled(err) {
		log.Debug(msg, zap.Error(err))
	}
	return status.Error(codes.Unavailable, "gRPC send failed")
}

func (a *agentAPI) logAndCapture(ctx context.Context, log *zap.Logger, msg string, err error) {
	// don't add logz.CorrelationIdFromContext(ctx) here as it's been added to the logger already
	log.Error(msg, zap.Error(err))
}

func (a *agentAPI) ToggleFeature(feature modagent.Feature, enabled bool) {
	a.featureTracker.ToggleFeature(feature, a.moduleName, enabled)
}

func (a *agentAPI) SubscribeToFeatureStatus(feature modagent.Feature, cb modagent.SubscribeCb) {
	a.featureTracker.Subscribe(feature, cb)
}

// Capture does nothing at the moment
func (a *agentAPI) Capture(err error, opts ...errortracking.CaptureOption) {
}

func (a *agentAPI) MakeGitLabRequest(ctx context.Context, path string, opts ...modagent.GitLabRequestOption) (*modagent.GitLabResponse, error) {
	config := modagent.ApplyRequestOptions(opts)
	ctx, cancel := context.WithCancel(ctx)
	client, errReq := a.client.MakeRequest(ctx)
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
		err := a.responseVisitor.Visit(client,
			grpctool.WithCallback(headerFieldNumber, func(header *grpctool.HttpResponse_Header) error {
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
			grpctool.WithCallback(dataFieldNumber, func(data *grpctool.HttpResponse_Data) error {
				_, err := pw.Write(data.Data)
				return err
			}),
			grpctool.WithCallback(trailerFieldNumber, func(trailer *grpctool.HttpResponse_Trailer) error {
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
	extra, err := anypb.New(&gitlab_access_rpc.HeaderExtra{
		ModuleName: a.moduleName,
	})
	if err != nil {
		return err
	}
	err = client.Send(&grpctool.HttpRequest{
		Message: &grpctool.HttpRequest_Header_{
			Header: &grpctool.HttpRequest_Header{
				Request: &prototool.HttpRequest{
					Method:  config.Method,
					Header:  prototool.HttpHeaderToValuesMap(config.Header),
					UrlPath: path,
					Query:   prototool.UrlValuesToValuesMap(config.Query),
				},
				Extra: extra,
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
				sendErr := client.Send(&grpctool.HttpRequest{
					Message: &grpctool.HttpRequest_Data_{
						Data: &grpctool.HttpRequest_Data{
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
	err = client.Send(&grpctool.HttpRequest{
		Message: &grpctool.HttpRequest_Trailer_{
			Trailer: &grpctool.HttpRequest_Trailer{},
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
