package agentk

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	managedObjectAnnotationName = "k8s-agent.gitlab.com/managed-object"
)

// synchronizerConfig holds configuration for a synchronizer.
type synchronizerConfig struct {
	log             *logrus.Entry
	projectId       string
	namespace       string
	kasClient       agentrpc.KasClient
	k8sClientGetter resource.RESTClientGetter
}

type resourceInfo struct {
	gcMark string
}

type synchronizer struct {
	ctx context.Context
	eng engine.GitOpsEngine
	synchronizerConfig
}

func (s *synchronizer) run() {
	req := &agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: s.projectId,
	}
	res, err := s.kasClient.GetObjectsToSynchronize(s.ctx, req)
	if err != nil {
		s.log.WithError(err).Warn("GetObjectsToSynchronize failed")
		return
	}
	for {
		objectsResp, err := res.Recv()
		if err != nil {
			switch {
			case err == io.EOF:
			case status.Code(err) == codes.DeadlineExceeded:
			case status.Code(err) == codes.Canceled:
			default:
				s.log.WithError(err).Warn("GetObjectsToSynchronize.Recv failed")
			}
			return
		}
		err = s.synchronize(objectsResp)
		if err != nil {
			s.log.WithError(err).Warn("Synchronization failed")
		}
	}
}

func (s *synchronizer) synchronize(objectsResp *agentrpc.ObjectsToSynchronizeResponse) error {
	objs, err := s.decodeObjectsToSynchronize(objectsResp.Objects)
	if err != nil {
		return err
	}
	markAsManaged(objs)
	result, err := s.eng.Sync(s.ctx, objs, s.isManaged, objectsResp.Revision, s.namespace)
	if err != nil {
		return fmt.Errorf("engine.Sync failed: %v", err)
	}
	for _, res := range result {
		s.log.WithFields(log.Fields{
			api.ResourceKey: res.ResourceKey.String(),
			api.SyncResult:  res.Message,
		}).Info("Synced")
	}
	return nil
}

func (s *synchronizer) isManaged(r *cache.Resource) bool {
	return r.Info.(*resourceInfo).gcMark == "managed" // TODO
}

func (s *synchronizer) decodeObjectsToSynchronize(objs []*agentrpc.ObjectToSynchronize) ([]*unstructured.Unstructured, error) {
	if len(objs) == 0 {
		return nil, nil
	}
	// TODO allow enforcing namespace
	builder := resource.NewBuilder(s.k8sClientGetter).
		ContinueOnError().
		Flatten().
		Local().
		Unstructured()
	for _, obj := range objs {
		builder.Stream(bytes.NewReader(obj.Object), obj.Source)
	}
	var res []*unstructured.Unstructured
	err := builder.Do().Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		un := info.Object.(*unstructured.Unstructured)
		// TODO enforce namespace is set for namespaced objects?
		res = append(res, un)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func markAsManaged(objs []*unstructured.Unstructured) {
	for _, obj := range objs {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string, 1)
		}
		annotations[managedObjectAnnotationName] = "managed" // TODO
		obj.SetAnnotations(annotations)
	}
}

func populateResourceInfoHandler(un *unstructured.Unstructured, isRoot bool) (interface{} /*info*/, bool /*cacheManifest*/) {
	// store gc mark of every resource
	gcMark := un.GetAnnotations()[managedObjectAnnotationName]
	// cache resources that has that mark to improve performance
	return &resourceInfo{
		gcMark: gcMark,
	}, gcMark != ""
}
