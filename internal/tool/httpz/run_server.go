package httpz

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

func RunServer(ctx context.Context, srv *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	var wg sync.WaitGroup
	defer wg.Wait() // wait for goroutine to shutdown active connections
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		if srv.Shutdown(shutdownCtx) != nil {
			srv.Close() // nolint: errcheck,gas,gosec
			// unhandled error above, but we are terminating anyway
		}
	}()

	err := srv.Serve(listener)

	if !errors.Is(err, http.ErrServerClosed) {
		// Failed to start or dirty shutdown
		return err
	}
	// Clean shutdown
	return nil
}
