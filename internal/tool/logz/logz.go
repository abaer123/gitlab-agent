package logz

// Do not add more dependencies to this package as it's depended upon by the whole codebase.

import (
	"context"
	"errors"
	"net"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/mask"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// These constants are for type-safe zap field helpers that are not here to:
// - avoid adding a dependency or
// - because they are not generally useful.
// Field names are here to make it possible to see all field names that are in use.
const (
	AgentConfig        = "agent_config"
	AgentFeatureName   = "feature_name"
	AgentFeatureStatus = "feature_status"
)

func NetAddressFromAddr(addr net.Addr) zap.Field {
	return NetAddress(addr.String())
}

func NetNetworkFromAddr(addr net.Addr) zap.Field {
	return NetNetwork(addr.Network())
}

func NetAddress(listenAddress string) zap.Field {
	return zap.String("net_address", listenAddress)
}

func NetNetwork(listenNetwork string) zap.Field {
	return zap.String("net_network", listenNetwork)
}

func IsWebSocket(isWebSocket bool) zap.Field {
	return zap.Bool("is_websocket", isWebSocket)
}

func AgentId(agentId int64) zap.Field {
	return zap.Int64("agent_id", agentId)
}

func CommitId(commitId string) zap.Field {
	return zap.String("commit_id", commitId)
}

func NumberOfFilesVisited(n uint32) zap.Field {
	return zap.Uint32("files_visited", n)
}
func NumberOfFilesSent(n uint32) zap.Field {
	return zap.Uint32("files_sent", n)
}

// The human-readable GitLab project path (e.g. gitlab-org/gitlab).
func ProjectId(projectId string) zap.Field {
	return zap.String("project_id", projectId)
}

func CorrelationIdFromContext(ctx context.Context) zap.Field {
	return CorrelationId(correlation.ExtractFromContext(ctx))
}

func CorrelationId(correlationId string) zap.Field {
	if correlationId == "" {
		return zap.Skip()
	}
	return zap.String(correlation.FieldName, correlationId)
}

func SentryDSN(sentryDSN string) zap.Field {
	return zap.String("sentry_dsn", maskURL(sentryDSN))
}

func SentryEnv(sentryEnv string) zap.Field {
	return zap.String("sentry_env", sentryEnv)
}

// Use for any keys in Redis.
func RedisKey(key []byte) zap.Field {
	return zap.Binary("redis_key", key)
}

// Use for any integer counters.
func U64Count(count uint64) zap.Field {
	return zap.Uint64("count", count)
}

// Use for any integer counters.
func TokenLimit(limit uint64) zap.Field {
	return zap.Uint64("token_limit", limit)
}

func RemovedHashKeys(n int) zap.Field {
	return zap.Int("removed_hash_keys", n)
}

// GitLab-kas or agentk module name.
func ModuleName(name string) zap.Field {
	return zap.String("mod_name", name)
}

func ConnectionId(connectionId int64) zap.Field {
	return zap.Int64("connection_id", connectionId)
}

func KasUrl(kasUrl string) zap.Field {
	return zap.String("kas_url", kasUrl)
}

func UrlPathPrefix(urlPrefix string) zap.Field {
	return zap.String("url_path_prefix", urlPrefix)
}

func UrlPath(url string) zap.Field {
	return zap.String("url_path", url)
}

func GrpcService(service string) zap.Field {
	return zap.String("grpc_service", service)
}

func GrpcMethod(method string) zap.Field {
	return zap.String("grpc_method", method)
}

func Error(err error) zap.Field {
	var errCorrelation errz.CorrelationError
	if errors.As(err, &errCorrelation) {
		return zap.Inline(correlationMarshaler{
			err:           err, // store the original error, not the unwrapped errCorrelation
			correlationId: errCorrelation.CorrelationId,
		})
	} else {
		return zap.Error(err) // nolint: forbidigo
	}
}

type correlationMarshaler struct {
	err           error
	correlationId string
}

func (m correlationMarshaler) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	zap.Error(m.err).AddTo(encoder)
	CorrelationId(m.correlationId).AddTo(encoder)
	return nil
}

// This should be in https://gitlab.com/gitlab-org/labkit/-/blob/master/mask/url.go
func maskURL(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return s
	}

	_, hasPassword := u.User.Password()

	if hasPassword || u.User.Username() != "" {
		u.User = url.User(mask.RedactionString)
	}

	return mask.URL(u.String())
}
