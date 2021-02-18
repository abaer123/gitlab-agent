package rpc

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	headersFieldNumber  protoreflect.FieldNumber = 1
	objectFieldNumber   protoreflect.FieldNumber = 2
	trailersFieldNumber protoreflect.FieldNumber = 3
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
	Log          *zap.Logger
	GitopsClient GitopsClient
	RetryPeriod  time.Duration
}

func (o *ObjectsToSynchronizeWatcher) Watch(ctx context.Context, req *ObjectsToSynchronizeRequest, callback ObjectsToSynchronizeCallback) {
	lastProcessedCommitId := req.CommitId
	sv, err := grpctool.NewStreamVisitor(&ObjectsToSynchronizeResponse{})
	if err != nil {
		// Coding error, must never happen
		panic(err)
	}
	retry.JitterUntil(ctx, o.RetryPeriod, func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // ensure streaming call is canceled
		req.CommitId = lastProcessedCommitId
		res, err := o.GitopsClient.GetObjectsToSynchronize(ctx, req)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				o.Log.Error("GetObjectsToSynchronize failed", zap.Error(err))
			}
			return
		}
		v := objectsToSynchronizeVisitor{}
		err = sv.Visit(res,
			grpctool.WithCallback(headersFieldNumber, v.OnHeaders),
			grpctool.WithCallback(objectFieldNumber, v.OnObject),
			grpctool.WithCallback(trailersFieldNumber, v.OnTrailers),
		)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				o.Log.Error("GetObjectsToSynchronize.Recv failed", zap.Error(err))
			}
			return
		}
		callback(ctx, v.objs)
		lastProcessedCommitId = v.objs.CommitId
	})
}

type objectsToSynchronizeVisitor struct {
	objs ObjectsToSynchronizeData
}

func (v *objectsToSynchronizeVisitor) OnHeaders(headers *ObjectsToSynchronizeResponse_Headers) error {
	v.objs.CommitId = headers.CommitId
	return nil
}

func (v *objectsToSynchronizeVisitor) OnObject(object *ObjectsToSynchronizeResponse_Object) error {
	lastIdx := len(v.objs.Sources) - 1
	if lastIdx >= 0 && v.objs.Sources[lastIdx].Name == object.Source {
		// Same source, append to the actual slice
		v.objs.Sources[lastIdx].Data = append(v.objs.Sources[lastIdx].Data, object.Data...)
	} else {
		// A new source
		v.objs.Sources = append(v.objs.Sources, ObjectSource{
			Name: object.Source,
			Data: object.Data,
		})
	}
	return nil
}

func (v *objectsToSynchronizeVisitor) OnTrailers(trailers *ObjectsToSynchronizeResponse_Trailers) error {
	// Nothing to do at the moment
	return nil
}
