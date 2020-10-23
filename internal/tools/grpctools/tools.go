package grpctools

import "context"

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
