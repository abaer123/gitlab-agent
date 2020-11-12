package grpctool

import (
	"context"
	"errors"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/errz"
	"google.golang.org/grpc/codes"
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
