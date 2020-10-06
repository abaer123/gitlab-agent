package agentk

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/sync"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_misc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	projectId = "bla123/bla-1"
	revision  = "rev12341234"
)

func TestGetObjectsToSynchronizeResumeConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	mockCtrl := gomock.NewController(t)
	mockEngineCtrl := gomock.NewController(t)
	engineCloser := mock_misc.NewMockCloser(mockCtrl)
	// engine is used concurrently with other mocks. So use a separate mock controller to avoid data races because
	// mock controllers are not thread safe.
	engine := mock_engine.NewMockGitOpsEngine(mockEngineCtrl)
	engineFactory := mock_engine.NewMockGitOpsEngineFactory(mockCtrl)
	kasClient := mock_agentrpc.NewMockKasClient(mockCtrl)
	stream1 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	job1started := make(chan struct{})
	stream2 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	gomock.InOrder(
		engineFactory.EXPECT().
			New(gomock.Any()).
			Return(engine),
		engine.EXPECT().
			Run().
			Return(engineCloser, nil),
		kasClient.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
			}), gomock.Any()).
			Return(stream1, nil),
		stream1.EXPECT().
			Recv().
			Return(&agentrpc.ObjectsToSynchronizeResponse{
				CommitId: revision,
			}, nil),
		stream1.EXPECT().
			Recv().
			Return(nil, io.EOF),
		kasClient.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				CommitId:  revision,
			}), gomock.Any()).
			Return(stream2, nil),
		stream2.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ObjectsToSynchronizeResponse, error) {
				<-job1started
				cancel()
				return nil, io.EOF
			}),
		engineCloser.EXPECT().
			Close().
			Return(nil),
	)
	engine.EXPECT().
		Sync(gomock.Any(), gomock.Len(0), gomock.Any(), revision, defaultNamespace, gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
			close(job1started) // signal that this job has been started
			<-ctx.Done()       // block until the job is cancelled
			return nil, ctx.Err()
		})

	d := &gitopsWorker{
		kasClient:                          kasClient,
		engineFactory:                      engineFactory,
		getObjectsToSynchronizeRetryPeriod: 10 * time.Millisecond,
		synchronizerConfig: synchronizerConfig{
			log: logrus.New().WithFields(nil),
			projectConfiguration: &agentcfg.ManifestProjectCF{
				Id:               projectId,
				DefaultNamespace: defaultNamespace, // as if user didn't specify configuration so it's the default value
			},
			k8sClientGetter: genericclioptions.NewTestConfigFlags(),
		},
	}
	d.Run(ctx)
}

func TestRunHappyPathNoObjects(t *testing.T) {
	s, engine, stream := setupWorker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := &agentrpc.ObjectsToSynchronizeResponse{
		CommitId: revision,
	}
	gomock.InOrder(
		stream.EXPECT().
			Recv().
			Return(resp, nil),
		stream.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	engine.EXPECT().
		Sync(gomock.Any(), gomock.Len(0), gomock.Any(), revision, defaultNamespace).
		DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
			cancel() // all good, stop run()
			return nil, nil
		})
	s.Run(ctx)
}

func TestRunHappyPath(t *testing.T) {
	s, engine, stream := setupWorker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	objs, resp := objsAndResp(t)
	gomock.InOrder(
		stream.EXPECT().
			Recv().
			Return(resp, nil),
		stream.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	engine.EXPECT().
		Sync(gomock.Any(), matcher.K8sObjectEq(t, objs, kube_testing.IgnoreAnnotation(managedObjectAnnotationName)), gomock.Any(), revision, defaultNamespace, gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
			cancel() // all good, stop run()
			return []common.ResourceSyncResult{
				{
					Status: common.ResultCodeSynced,
				},
			}, nil
		})

	s.Run(ctx)
}

func TestRunHappyPathSyncCancellation(t *testing.T) {
	s, engine, stream := setupWorker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	objs, resp1 := objsAndResp(t)
	resp2 := &agentrpc.ObjectsToSynchronizeResponse{
		CommitId: revision,
	}
	job1started := make(chan struct{})
	gomock.InOrder(
		stream.EXPECT().
			Recv().
			Return(resp1, nil),
		stream.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ObjectsToSynchronizeResponse, error) {
				<-job1started
				return resp2, nil
			}),
		stream.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	gomock.InOrder(
		engine.EXPECT().
			Sync(gomock.Any(), matcher.K8sObjectEq(t, objs, kube_testing.IgnoreAnnotation(managedObjectAnnotationName)), gomock.Any(), revision, defaultNamespace, gomock.Any()).
			DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
				close(job1started) // signal that this job has been started
				<-ctx.Done()       // block until the job is cancelled
				return nil, ctx.Err()
			}),
		engine.EXPECT().
			Sync(gomock.Any(), gomock.Len(0), gomock.Any(), revision, defaultNamespace, gomock.Any()).
			DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
				cancel() // all good, stop run()
				return nil, nil
			}),
	)

	s.Run(ctx)
}

func objsAndResp(t *testing.T) ([]*unstructured.Unstructured, *agentrpc.ObjectsToSynchronizeResponse) {
	objs := []*unstructured.Unstructured{
		kube_testing.ToUnstructured(t, testNs1()),
		kube_testing.ToUnstructured(t, testMap1()),
		kube_testing.ToUnstructured(t, testMap2()),
	}
	resp1 := &agentrpc.ObjectsToSynchronizeResponse{
		CommitId: revision,
		Objects: []*agentrpc.ObjectToSynchronize{
			{
				// Multi-document YAML
				Object: kube_testing.ObjsToYAML(t, objs[0], objs[1]),
			},
			{
				// Single ConfigMap object
				Object: kube_testing.ObjsToYAML(t, objs[2]),
			},
		},
	}
	return objs, resp1
}

func setupWorker(t *testing.T) (*gitopsWorker, *mock_engine.MockGitOpsEngine, *mock_agentrpc.MockKas_GetObjectsToSynchronizeClient) {
	mockCtrl := gomock.NewController(t)
	mockEngineCtrl := gomock.NewController(t)
	engineCloser := mock_misc.NewMockCloser(mockCtrl)
	// engine is used concurrently with other mocks. So use a separate mock controller to avoid data races because
	// mock controllers are not thread safe.
	engine := mock_engine.NewMockGitOpsEngine(mockEngineCtrl)
	engineFactory := mock_engine.NewMockGitOpsEngineFactory(mockCtrl)
	kasClient := mock_agentrpc.NewMockKasClient(mockCtrl)
	stream := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	gomock.InOrder(
		engineFactory.EXPECT().
			New(gomock.Any()).
			Return(engine),
		engine.EXPECT().
			Run().
			Return(engineCloser, nil),
		kasClient.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
			}), gomock.Any()).
			Return(stream, nil),
		engineCloser.EXPECT().
			Close().
			Return(nil),
	)
	d := &gitopsWorker{
		kasClient:                          kasClient,
		engineFactory:                      engineFactory,
		getObjectsToSynchronizeRetryPeriod: 10 * time.Millisecond,
		synchronizerConfig: synchronizerConfig{
			log: logrus.New().WithFields(nil),
			projectConfiguration: &agentcfg.ManifestProjectCF{
				Id:               projectId,
				DefaultNamespace: defaultNamespace, // as if user didn't specify configuration so it's the default value
			},
			k8sClientGetter: genericclioptions.NewTestConfigFlags(),
		},
	}
	return d, engine, stream
}

func testMap1() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "map1",
			Namespace: "test1",
			Annotations: map[string]string{
				"k1": "v1",
			},
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}
}

func testMap2() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "map2",
			Namespace: "test2",
			Annotations: map[string]string{
				"k2": "v2",
			},
		},
		Data: map[string]string{
			"key2": "value2",
		},
	}
}

func testNs1() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns1",
			Annotations: map[string]string{
				"k3": "v3",
			},
		},
	}
}
