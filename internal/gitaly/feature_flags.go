package gitaly

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func appendFeatureFlagsToContext(ctx context.Context, features map[string]string) context.Context {
	if len(features) == 0 {
		return ctx
	}
	kv := make([]string, 0, len(features)*2)
	for k, v := range features {
		kv = append(kv, k, v)
	}
	return metadata.AppendToOutgoingContext(ctx, kv...)
}
