package modshared

import (
	"context"

	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
)

// API provides the API for the module to use.
type API interface {
	errortracking.Tracker
	// HandleProcessingError can be used to handle errors occurring while processing a request.
	// If err is a (or wraps a) errz.UserError, it might be handled specially.
	HandleProcessingError(ctx context.Context, log *zap.Logger, msg string, err error)
	// HandleSendError can be used to handle error produced by gRPC Send() or SendMsg() method.
	// It returns an error, compatible with gRPC status package.
	HandleSendError(log *zap.Logger, msg string, err error) error
}
