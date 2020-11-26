package agentrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"go.uber.org/zap"
)

type nextObjectStreamStep int

const (
	expectingHeaders nextObjectStreamStep = iota
	expectingObjectOrTrailers
	expectingEOF
)

type ObjectSource struct {
	Name string
	Data []byte
}

type ObjectsToSynchronizeData struct {
	CommitId string
	Sources  []ObjectSource
}

type ObjectsToSynchronizeCallback func(context.Context, ObjectsToSynchronizeData)

// ObjectsToSynchronizeWatcherInterface abstracts ObjectsToSynchronizeWatcher.
type ObjectsToSynchronizeWatcherInterface interface {
	Watch(context.Context, *ObjectsToSynchronizeRequest, ObjectsToSynchronizeCallback)
}

type ObjectsToSynchronizeWatcher struct {
	Log         *zap.Logger
	KasClient   KasClient
	RetryPeriod time.Duration
}

func (o *ObjectsToSynchronizeWatcher) Watch(ctx context.Context, req *ObjectsToSynchronizeRequest, callback ObjectsToSynchronizeCallback) {
	lastProcessedCommitId := req.CommitId
	retry.JitterUntil(ctx, o.RetryPeriod, func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // ensure streaming call is canceled
		req.CommitId = lastProcessedCommitId
		res, err := o.KasClient.GetObjectsToSynchronize(ctx, req)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				o.Log.Error("GetObjectsToSynchronize failed", zap.Error(err))
			}
			return
		}
		var (
			objs             ObjectsToSynchronizeData
			headersReceived  bool
			trailersReceived bool
		)
		expecting := expectingHeaders
	objectStream:
		for {
			objectsResp, err := res.Recv()
			if err != nil {
				switch {
				case errors.Is(err, io.EOF):
					break objectStream
				case grpctool.RequestCanceled(err):
				default:
					o.Log.Error("GetObjectsToSynchronize.Recv failed", zap.Error(err))
				}
				return
			}
			if expecting == expectingEOF {
				o.Log.Error("GetObjectsToSynchronize.Recv - unexpected data after trailers")
				return
			}
			switch msg := objectsResp.Message.(type) {
			case *ObjectsToSynchronizeResponse_Headers_:
				if expecting != expectingHeaders {
					o.Log.Error("GetObjectsToSynchronize.Recv - expecting object or trailers, got headers")
					return
				}
				headersReceived = true
				expecting = expectingObjectOrTrailers
				objs.CommitId = msg.Headers.CommitId
			case *ObjectsToSynchronizeResponse_Object_:
				if expecting != expectingObjectOrTrailers {
					o.Log.Error("GetObjectsToSynchronize.Recv - expecting headers, got object")
					return
				}
				lastIdx := len(objs.Sources) - 1
				object := msg.Object
				if lastIdx >= 0 && objs.Sources[lastIdx].Name == object.Source {
					// Same source, append to the actual slice
					objs.Sources[lastIdx].Data = append(objs.Sources[lastIdx].Data, object.Data...)
					continue
				}
				objs.Sources = append(objs.Sources, ObjectSource{
					Name: object.Source,
					Data: object.Data,
				})
			case *ObjectsToSynchronizeResponse_Trailers_:
				if expecting != expectingObjectOrTrailers {
					o.Log.Error("GetObjectsToSynchronize.Recv - expecting headers, got trailers")
					return
				}
				trailersReceived = true
				expecting = expectingEOF
			default:
				o.Log.Error(fmt.Sprintf("GetObjectsToSynchronize.Recv returned an unexpected type: %T", objectsResp.Message))
				return
			}
		}
		switch {
		case !headersReceived && !trailersReceived:
		case headersReceived && trailersReceived:
			// All good, work on received state
			callback(ctx, objs)
			lastProcessedCommitId = objs.CommitId
		default:
			// This should never happen.
			o.Log.Error(fmt.Sprintf(
				"Server didn't send both headers (%t) and trailers (%t) for objects stream. It's a bug! Number of sources received: %d, commit id: %s",
				headersReceived,
				trailersReceived,
				len(objs.Sources),
				objs.CommitId),
			)
		}
	})
}
