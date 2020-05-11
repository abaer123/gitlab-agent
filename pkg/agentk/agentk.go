package agentk

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ash2k/stager/wait"
	"gitlab.com/ash2k/gitlab-agent/agentrpc"
	"k8s.io/client-go/rest"
)

type Agent struct {
	Client     agentrpc.ReverseProxyServiceClient
	RestClient rest.Interface
}

func (a *Agent) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		stream, err := a.Client.GetRequests(ctx)
		if err != nil {
			// TODO log, retry with backoff
			fmt.Print(err)
			time.Sleep(1 * time.Second)
			continue
		}
		a.handleStream(ctx, stream)
	}
}

func (a *Agent) handleStream(ctx context.Context, stream agentrpc.ReverseProxyService_GetRequestsClient) {
	defer stream.CloseSend()
	var wg wait.Group
	defer wg.Wait() // wait for all the running handlers to finish
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // cancel all the running handlers
	for {
		req, err := stream.Recv()
		switch err {
		case io.EOF: // End of the stream
			// Wait for all the running handlers to finish.
			// Do not exit the loop, block here to avoid cancelling the running handlers, let them finish.
			wg.Wait()
			return
		case nil: // No error, handle the request
			handler := reverseRequestHandler{
				req: req,
			}
			wg.StartWithContext(ctx, handler.Handle)
		default: // Some error, steam is aborted
			return
		}
	}
}
