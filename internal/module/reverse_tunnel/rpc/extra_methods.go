package rpc

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"google.golang.org/grpc/metadata"
)

func (x *RequestInfo) Metadata() metadata.MD {
	return grpctool.ValuesMapToMeta(x.Meta)
}

func (x *Header) Metadata() metadata.MD {
	return grpctool.ValuesMapToMeta(x.Meta)
}

func (x *Trailer) Metadata() metadata.MD {
	return grpctool.ValuesMapToMeta(x.Meta)
}
