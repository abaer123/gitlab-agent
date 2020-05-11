package agentk

import (
	"context"

	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
	"k8s.io/client-go/rest"
)

type reverseRequestHandler struct {
	req       *agentrpc.KubernetesRequest
	responder agentrpc.ReverseProxyService_GetRequestsClient
	rest      rest.Interface
}

func (r *reverseRequestHandler) Handle(ctx context.Context) {
	//req := r.rest.Verb(r.req.Verb)

}
