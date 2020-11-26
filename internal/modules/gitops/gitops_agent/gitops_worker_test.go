package gitops_agent

import (
	"context"
	"testing"

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

var (
	_ GitOpsEngineFactory = &DefaultGitOpsEngineFactory{}
)

func TestRunHappyPathNoObjects(t *testing.T) {
	w, engine, watcher := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.synchronizerConfig.projectConfiguration.Paths,
	}
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
			DoAndReturn(func(ctx context.Context, req *agentrpc.ObjectsToSynchronizeRequest, callback agentrpc.ObjectsToSynchronizeCallback) {
				callback(ctx, agentrpc.ObjectsToSynchronizeData{
					CommitId: revision,
				})
				<-ctx.Done()
			}),
		engine.EXPECT().
			Sync(gomock.Any(), gomock.Len(0), gomock.Any(), revision, defaultNamespace, gomock.Any()).
			DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
				cancel() // all good, stop run()
				return nil, nil
			}),
	)
	w.Run(ctx)
}

func TestRunHappyPath(t *testing.T) {
	w, engine, watcher := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.synchronizerConfig.projectConfiguration.Paths,
	}
	objs := []*unstructured.Unstructured{
		kube_testing.ToUnstructured(t, testMap1()),
		kube_testing.ToUnstructured(t, testNs1()),
		kube_testing.ToUnstructured(t, testMap2()),
	}
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
			DoAndReturn(func(ctx context.Context, req *agentrpc.ObjectsToSynchronizeRequest, callback agentrpc.ObjectsToSynchronizeCallback) {
				callback(ctx, agentrpc.ObjectsToSynchronizeData{
					CommitId: revision,
					Sources: []agentrpc.ObjectSource{
						{
							Name: "obj1.yaml",
							Data: kube_testing.ObjsToYAML(t, objs[0]),
						},
						{
							Name: "obj2.yaml",
							Data: kube_testing.ObjsToYAML(t, objs[1], objs[2]),
						},
					},
				})
				<-ctx.Done()
			}),
		engine.EXPECT().
			Sync(gomock.Any(), matcher.K8sObjectEq(t, objs, kube_testing.IgnoreAnnotation(managedObjectAnnotationName)), gomock.Any(), revision, defaultNamespace, gomock.Any()).
			DoAndReturn(func(ctx context.Context, resources []*unstructured.Unstructured, isManaged func(r *cache.Resource) bool, revision string, namespace string, opts ...sync.SyncOpt) ([]common.ResourceSyncResult, error) {
				cancel() // all good, stop run()
				return []common.ResourceSyncResult{
					{
						Status: common.ResultCodeSynced,
					},
				}, nil
			}),
	)
	w.Run(ctx)
}

func TestRunHappyPathSyncCancellation(t *testing.T) {
	w, engine, watcher := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.synchronizerConfig.projectConfiguration.Paths,
	}
	objs := []*unstructured.Unstructured{
		kube_testing.ToUnstructured(t, testMap1()),
		kube_testing.ToUnstructured(t, testNs1()),
		kube_testing.ToUnstructured(t, testMap2()),
	}
	job1started := make(chan struct{})
	watcher.EXPECT().
		Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *agentrpc.ObjectsToSynchronizeRequest, callback agentrpc.ObjectsToSynchronizeCallback) {
			callback(ctx, agentrpc.ObjectsToSynchronizeData{
				CommitId: revision,
				Sources: []agentrpc.ObjectSource{
					{
						Name: "obj1.yaml",
						Data: kube_testing.ObjsToYAML(t, objs[0]),
					},
					{
						Name: "obj2.yaml",
						Data: kube_testing.ObjsToYAML(t, objs[1], objs[2]),
					},
				},
			})
			<-job1started
			callback(ctx, agentrpc.ObjectsToSynchronizeData{
				CommitId: revision,
			})
			<-ctx.Done()
		})
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
	w.Run(ctx)
}

func setupWorker(t *testing.T) (*gitopsWorker, *mock_engine.MockGitOpsEngine, *mock_agentrpc.MockObjectsToSynchronizeWatcherInterface) {
	mockCtrl := gomock.NewController(t)
	mockEngineCtrl := gomock.NewController(t)
	// engine is used concurrently with other mocks. So use a separate mock controller to avoid data races because
	// mock controllers are not thread safe.
	engine := mock_engine.NewMockGitOpsEngine(mockEngineCtrl)
	engineFactory := mock_engine.NewMockGitOpsEngineFactory(mockCtrl)
	watcher := mock_agentrpc.NewMockObjectsToSynchronizeWatcherInterface(mockCtrl)
	engineWasStopped := false
	t.Cleanup(func() {
		assert.True(t, engineWasStopped)
	})
	gomock.InOrder(
		engineFactory.EXPECT().
			New(gomock.Any(), gomock.Any()).
			Return(engine),
		engine.EXPECT().
			Run().
			Return(func() {
				engineWasStopped = true
			}, nil),
	)
	w := &gitopsWorker{
		objWatcher:    watcher,
		engineFactory: engineFactory,
		synchronizerConfig: synchronizerConfig{
			log: zaptest.NewLogger(t),
			projectConfiguration: &agentcfg.ManifestProjectCF{
				Id:               projectId,
				DefaultNamespace: defaultNamespace, // as if user didn't specify configuration so it's the default value
				Paths: []*agentcfg.PathCF{
					{
						Glob: "*.yaml",
					},
				},
			},
			k8sClientGetter: genericclioptions.NewTestConfigFlags(),
		},
	}
	return w, engine, watcher
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
