package agentg

import "gitlab.com/ash2k/gitlab-agent/agentrpc"

type Agent struct {
}

func (a *Agent) GetRequests(stream agentrpc.ReverseProxyService_GetRequestsServer) error {
	return nil
}
