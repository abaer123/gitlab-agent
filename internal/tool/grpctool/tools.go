package grpctool

import (
	"context"
	"errors"
	"net"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	grpccorrelation "gitlab.com/gitlab-org/labkit/correlation/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RequestCanceled(err error) bool {
	if errz.ContextDone(err) {
		return true
	}
	for err != nil {
		code := status.Code(err)
		if code == codes.Canceled || code == codes.DeadlineExceeded {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

func StartServer(stage stager.Stage, server *grpc.Server, listener func() (net.Listener, error)) {
	stage.Go(func(ctx context.Context) error {
		// gRPC listener
		lis, err := listener()
		if err != nil {
			return err
		}
		return server.Serve(lis)
	})
	stage.Go(func(ctx context.Context) error {
		<-ctx.Done() // can be cancelled because Serve() failed or main ctx was canceled or some stage failed
		server.GracefulStop()
		return nil
	})
}

func IsStatusError(err error) bool {
	_, ok := err.(interface { // nolint:errorlint
		GRPCStatus() *status.Status
	})
	return ok
}

func MetaToValuesMap(meta metadata.MD) map[string]*prototool.Values {
	if len(meta) == 0 {
		return nil
	}
	result := make(map[string]*prototool.Values, len(meta))
	for k, v := range meta {
		val := make([]string, len(v))
		copy(val, v) // metadata may be mutated, so copy
		result[k] = &prototool.Values{
			Value: val,
		}
	}
	return result
}

func ValuesMapToMeta(vals map[string]*prototool.Values) metadata.MD {
	result := make(metadata.MD, len(vals))
	for k, v := range vals {
		val := make([]string, len(v.Value))
		copy(val, v.Value) // metadata may be mutated, so copy
		result[k] = val
	}
	return result
}

func MaybeWrapWithCorrelationId(err error, client grpc.ClientStream) error {
	md, headerErr := client.Header()
	if headerErr != nil {
		return err
	}
	return errz.MaybeWrapWithCorrelationId(err, grpccorrelation.CorrelationIDFromMetadata(md))
}

func DeferMaybeWrapWithCorrelationId(err *error, client grpc.ClientStream) {
	if *err == nil {
		return
	}
	*err = MaybeWrapWithCorrelationId(*err, client)
}
