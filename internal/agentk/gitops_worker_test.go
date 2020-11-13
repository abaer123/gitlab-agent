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
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	projectId        = "bla123/bla-1"
	revision         = "rev12341234"
	defaultNamespace = "testing1"
)

func TestGetObjectsToSynchronizeResumeConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	mockEngineCtrl := gomock.NewController(t)
	// engine is used concurrently with other mocks. So use a separate mock controller to avoid data races because
	// mock controllers are not thread safe.
	engine := mock_engine.NewMockGitOpsEngine(mockEngineCtrl)
	engineFactory := mock_engine.NewMockGitOpsEngineFactory(mockCtrl)
	kasClient := mock_agentrpc.NewMockKasClient(mockCtrl)
	stream1 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	job1started := make(chan struct{})
	stream2 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	engineWasStopped := false
	defer func() {
		assert.True(t, engineWasStopped)
	}()
	pathsCfg := []*agentcfg.PathCF{
		{
			Glob: "*.yaml",
		},
	}
	gomock.InOrder(
		engineFactory.EXPECT().
			New(gomock.Any(), gomock.Any()).
			Return(engine),
		engine.EXPECT().
			Run().
			Return(func() {
				engineWasStopped = true
			}, nil),
		kasClient.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				Paths:     pathsCfg,
			}), gomock.Any()).
			Return(stream1, nil),
		stream1.EXPECT().
			Recv().
			Return(&agentrpc.ObjectsToSynchronizeResponse{
				Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
					Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
						CommitId: revision,
					},
				},
			}, nil),
		stream1.EXPECT().
			Recv().
			Return(&agentrpc.ObjectsToSynchronizeResponse{
				Message: &agentrpc.ObjectsToSynchronizeResponse_Trailers_{
					Trailers: &agentrpc.ObjectsToSynchronizeResponse_Trailers{},
				},
			}, nil),
		stream1.EXPECT().
			Recv().
			Return(nil, io.EOF),
		kasClient.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				CommitId:  revision,
				Paths:     pathsCfg,
			}), gomock.Any()).
			Return(stream2, nil),
		stream2.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ObjectsToSynchronizeResponse, error) {
				<-job1started
				cancel()
				return nil, io.EOF
			}),
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
		getObjectsToSynchronizeRetryPeriod: 10 * time.Millisecond, // must be small, to retry fast
		synchronizerConfig: synchronizerConfig{
			log: zaptest.NewLogger(t),
			projectConfiguration: &agentcfg.ManifestProjectCF{
				Id:               projectId,
				DefaultNamespace: defaultNamespace, // as if user didn't specify configuration so it's the default value
				Paths:            pathsCfg,
			},
			k8sClientGetter: genericclioptions.NewTestConfigFlags(),
		},
	}
	d.Run(ctx)
}

func TestRunHappyPathNoObjects(t *testing.T) {
	_, s, engine, stream, _ := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	headers := &agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
			Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
				CommitId: revision,
			},
		},
	}
	trailers := &agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Trailers_{
			Trailers: &agentrpc.ObjectsToSynchronizeResponse_Trailers{},
		},
	}
	gomock.InOrder(
		stream.EXPECT().
			Recv().
			Return(headers, nil),
		stream.EXPECT().
			Recv().
			Return(trailers, nil),
		stream.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	engine.EXPECT().
		Sync(gomock.Any(), gomock.Len(0), gomock.Any(), revision, defaultNamespace, gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
			cancel() // all good, stop run()
			return nil, nil
		})
	s.Run(ctx)
}

func TestRunHappyPath(t *testing.T) {
	_, s, engine, stream, _ := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	objs, headers, resp1, resp2, resp3, trailers := objsAndResp(t)
	gomock.InOrder(
		stream.EXPECT().
			Recv().
			Return(headers, nil),
		stream.EXPECT().
			Recv().
			Return(resp1, nil),
		stream.EXPECT().
			Recv().
			Return(resp2, nil),
		stream.EXPECT().
			Recv().
			Return(resp3, nil),
		stream.EXPECT().
			Recv().
			Return(trailers, nil),
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
	mockCtrl, s, engine, stream1, kasClient := setupWorker(t)
	s.getObjectsToSynchronizeRetryPeriod = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	objs, headers, resp1, resp2, resp3, trailers := objsAndResp(t)
	job1started := make(chan struct{})
	stream2 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	gomock.InOrder(
		stream1.EXPECT().
			Recv().
			Return(headers, nil),
		stream1.EXPECT().
			Recv().
			Return(resp1, nil),
		stream1.EXPECT().
			Recv().
			Return(resp2, nil),
		stream1.EXPECT().
			Recv().
			Return(resp3, nil),
		stream1.EXPECT().
			Recv().
			Return(trailers, nil),
		stream1.EXPECT().
			Recv().
			Return(nil, io.EOF),
		kasClient.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				CommitId:  revision,
				Paths:     s.projectConfiguration.Paths,
			}), gomock.Any()).
			DoAndReturn(func(ctx context.Context, req *agentrpc.ObjectsToSynchronizeRequest, opts ...grpc.CallOption) (agentrpc.Kas_GetObjectsToSynchronizeClient, error) {
				<-job1started
				return stream2, nil
			}),
		stream2.EXPECT().
			Recv().
			Return(headers, nil),
		stream2.EXPECT().
			Recv().
			Return(trailers, nil),
		stream2.EXPECT().
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

func objsAndResp(t *testing.T) ([]*unstructured.Unstructured, *agentrpc.ObjectsToSynchronizeResponse, *agentrpc.ObjectsToSynchronizeResponse, *agentrpc.ObjectsToSynchronizeResponse, *agentrpc.ObjectsToSynchronizeResponse, *agentrpc.ObjectsToSynchronizeResponse) {
	objs := []*unstructured.Unstructured{
		kube_testing.ToUnstructured(t, testMap1()),
		kube_testing.ToUnstructured(t, testNs1()),
		kube_testing.ToUnstructured(t, testMap2()),
	}
	headers := &agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
			Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
				CommitId: revision,
			},
		},
	}
	// Single ConfigMap object
	resp1 := &agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Object_{
			Object: &agentrpc.ObjectsToSynchronizeResponse_Object{
				Source: "obj1.yaml",
				Data:   kube_testing.ObjsToYAML(t, objs[0]),
			},
		},
	}
	// Multi-document YAML
	data2 := kube_testing.ObjsToYAML(t, objs[1], objs[2])
	resp2 := &agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Object_{
			Object: &agentrpc.ObjectsToSynchronizeResponse_Object{
				Source: "obj2.yaml",
				Data:   data2[:2], // first part
			},
		},
	}
	resp3 := &agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Object_{
			Object: &agentrpc.ObjectsToSynchronizeResponse_Object{
				Source: "obj2.yaml",
				Data:   data2[2:], // last part
			},
		},
	}
	trailers := &agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Trailers_{
			Trailers: &agentrpc.ObjectsToSynchronizeResponse_Trailers{},
		},
	}
	return objs, headers, resp1, resp2, resp3, trailers
}

func setupWorker(t *testing.T) (*gomock.Controller, *gitopsWorker, *mock_engine.MockGitOpsEngine, *mock_agentrpc.MockKas_GetObjectsToSynchronizeClient, *mock_agentrpc.MockKasClient) {
	mockCtrl := gomock.NewController(t)
	mockEngineCtrl := gomock.NewController(t)
	// engine is used concurrently with other mocks. So use a separate mock controller to avoid data races because
	// mock controllers are not thread safe.
	engine := mock_engine.NewMockGitOpsEngine(mockEngineCtrl)
	engineFactory := mock_engine.NewMockGitOpsEngineFactory(mockCtrl)
	kasClient := mock_agentrpc.NewMockKasClient(mockCtrl)
	stream := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	engineWasStopped := false
	t.Cleanup(func() {
		assert.True(t, engineWasStopped)
	})
	pathsCfg := []*agentcfg.PathCF{
		{
			Glob: "*.yaml",
		},
	}
	gomock.InOrder(
		engineFactory.EXPECT().
			New(gomock.Any(), gomock.Any()).
			Return(engine),
		engine.EXPECT().
			Run().
			Return(func() {
				engineWasStopped = true
			}, nil),
		kasClient.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				Paths:     pathsCfg,
			}), gomock.Any()).
			Return(stream, nil),
	)
	d := &gitopsWorker{
		kasClient:                          kasClient,
		engineFactory:                      engineFactory,
		getObjectsToSynchronizeRetryPeriod: 10 * time.Second,
		synchronizerConfig: synchronizerConfig{
			log: zaptest.NewLogger(t),
			projectConfiguration: &agentcfg.ManifestProjectCF{
				Id:               projectId,
				DefaultNamespace: defaultNamespace, // as if user didn't specify configuration so it's the default value
				Paths:            pathsCfg,
			},
			k8sClientGetter: genericclioptions.NewTestConfigFlags(),
		},
	}
	return mockCtrl, d, engine, stream, kasClient
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
