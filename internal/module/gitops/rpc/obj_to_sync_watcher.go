package rpc

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	headerFieldNumber  protoreflect.FieldNumber = 1
	objectFieldNumber  protoreflect.FieldNumber = 2
	trailerFieldNumber protoreflect.FieldNumber = 3
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
	Backoff      retry.BackoffManagerFactory
}

func (o *ObjectsToSynchronizeWatcher) Watch(ctx context.Context, req *ObjectsToSynchronizeRequest, callback ObjectsToSynchronizeCallback) {
	sv, err := grpctool.NewStreamVisitor(&ObjectsToSynchronizeResponse{})
	if err != nil {
		// Coding error, must never happen
		panic(err)
	}
	lastProcessedCommitId := req.CommitId
	_ = retry.PollWithBackoff(ctx, o.Backoff(), true, 0 /* we never use Continue, so doesn't matter */, func() (error, retry.AttemptResult) {
		ctx, cancel := context.WithCancel(ctx) // nolint:govet
		defer cancel()                         // ensure streaming call is canceled
		req.CommitId = lastProcessedCommitId
		res, err := o.GitopsClient.GetObjectsToSynchronize(ctx, req)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				o.Log.Error("GetObjectsToSynchronize failed", zap.Error(err))
			}
			return nil, retry.Backoff
		}
		v := objectsToSynchronizeVisitor{}
		err = sv.Visit(res,
			grpctool.WithCallback(headerFieldNumber, v.OnHeader),
			grpctool.WithCallback(objectFieldNumber, v.OnObject),
			grpctool.WithCallback(trailerFieldNumber, v.OnTrailer),
		)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				o.Log.Error("GetObjectsToSynchronize.Recv failed", zap.Error(err))
			}
			return nil, retry.Backoff
		}
		if !v.nonEmptyStream {
			// Server closed the stream without sending us anything.
			// It's fine, will just reopen the connection.
			return nil, retry.ContinueImmediately
		}
		callback(ctx, v.objs)
		lastProcessedCommitId = v.objs.CommitId
		return nil, retry.ContinueImmediately
	})
}

type objectsToSynchronizeVisitor struct {
	objs           ObjectsToSynchronizeData
	nonEmptyStream bool
}

func (v *objectsToSynchronizeVisitor) OnHeader(header *ObjectsToSynchronizeResponse_Header) error {
	v.objs.CommitId = header.CommitId
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

func (v *objectsToSynchronizeVisitor) OnTrailer(trailer *ObjectsToSynchronizeResponse_Trailer) error {
	v.nonEmptyStream = true
	return nil
}
