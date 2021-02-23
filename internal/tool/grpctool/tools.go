package grpctool

import (
	"context"
	"errors"
	"net"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// JoinContexts returns a ContextAugmenter that alters the main context to propagate cancellation from the aux context.
// The returned context uses main as the parent context to inherit attached values.
//
// This helper is used here to propagate done signal from both gRPC stream's context (stream is closed/broken) and
// main program's context (program needs to stop). Polling should stop when one of this conditions happens so using
// only one of these two contexts is not good enough.
func JoinContexts(aux context.Context) ContextAugmenter {
	return func(main context.Context) (context.Context, error) {
		ctx, cancel := context.WithCancel(main)
		go func() {
			defer cancel()
			select {
			case <-main.Done():
			case <-aux.Done():
			}
		}()
		return ctx, nil
	}
}

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

func StartServer(stage stager.Stage, server *grpc.Server, interceptorsCancel context.CancelFunc, listener func() (net.Listener, error)) {
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
		interceptorsCancel()
		server.GracefulStop()
		return nil
	})
}

func IsStatusError(err error) bool {
	_, ok := err.(interface {
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
