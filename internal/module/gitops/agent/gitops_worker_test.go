package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

const (
	projectId        = "bla123/bla-1"
	revision         = "rev12341234"
	defaultNamespace = "testing1"
)

var (
	_ GitopsWorker        = &defaultGitopsWorker{}
	_ GitopsWorkerFactory = &defaultGitopsWorkerFactory{}
)

func TestRun_HappyPath_NoObjects(t *testing.T) {
	w, applier, watcher := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.project.Paths,
	}
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
			Do(func(ctx context.Context, req *rpc.ObjectsToSynchronizeRequest, callback rpc.ObjectsToSynchronizeCallback) {
				callback(ctx, rpc.ObjectsToSynchronizeData{
					CommitId: revision,
				})
				<-ctx.Done()
			}),
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), gomock.Len(0), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				cancel() // all good, stop run()
				c := make(chan event.Event)
				close(c)
				return c
			}),
	)
	w.Run(ctx)
}

func TestRun_HappyPath_NoInventoryTemplate(t *testing.T) {
	w, applier, watcher := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.project.Paths,
	}
	objs := []*unstructured.Unstructured{
		kube_testing.ToUnstructured(t, testMap1()),
		kube_testing.ToUnstructured(t, testNs1()),
		kube_testing.ToUnstructured(t, testMap2()),
	}
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
			Do(func(ctx context.Context, req *rpc.ObjectsToSynchronizeRequest, callback rpc.ObjectsToSynchronizeCallback) {
				callback(ctx, rpc.ObjectsToSynchronizeData{
					CommitId: revision,
					Sources: []rpc.ObjectSource{
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
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), matcher.K8sObjectEq(t, objs), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				assert.Equal(t, w.project.DefaultNamespace, invInfo.Namespace())
				cancel() // all good, stop Run()
				c := make(chan event.Event)
				close(c)
				return c
			}),
	)
	w.Run(ctx)
}

func TestRun_HappyPath_InventoryTemplate(t *testing.T) {
	w, applier, watcher := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.project.Paths,
	}
	inv := invObject("some_id", "some_ns")
	objs := []*unstructured.Unstructured{kube_testing.ToUnstructured(t, testMap1())}
	gomock.InOrder(
		watcher.EXPECT().
			Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
			Do(func(ctx context.Context, req *rpc.ObjectsToSynchronizeRequest, callback rpc.ObjectsToSynchronizeCallback) {
				callback(ctx, rpc.ObjectsToSynchronizeData{
					CommitId: revision,
					Sources: []rpc.ObjectSource{
						{
							Name: "obj1.yaml",
							Data: kube_testing.ObjsToYAML(t, objs[0], inv),
						},
					},
				})
				<-ctx.Done()
			}),
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), matcher.K8sObjectEq(t, objs), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				assert.Equal(t, "some_ns", invInfo.Namespace())
				assert.Equal(t, "inventory-some_id", invInfo.Name())
				cancel() // all good, stop Run()
				c := make(chan event.Event)
				close(c)
				return c
			}),
	)
	w.Run(ctx)
}

func TestRun_SyncCancellation(t *testing.T) {
	w, applier, watcher := setupWorker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.project.Paths,
	}
	objs := []*unstructured.Unstructured{
		kube_testing.ToUnstructured(t, testMap1()),
		kube_testing.ToUnstructured(t, testNs1()),
		kube_testing.ToUnstructured(t, testMap2()),
	}
	job1started := make(chan struct{})
	watcher.EXPECT().
		Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
		Do(func(ctx context.Context, req *rpc.ObjectsToSynchronizeRequest, callback rpc.ObjectsToSynchronizeCallback) {
			callback(ctx, rpc.ObjectsToSynchronizeData{
				CommitId: revision,
				Sources: []rpc.ObjectSource{
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
			callback(ctx, rpc.ObjectsToSynchronizeData{
				CommitId: revision,
			})
			<-ctx.Done()
		})
	gomock.InOrder(
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), matcher.K8sObjectEq(t, objs), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				close(job1started) // signal that this job has been started
				c := make(chan event.Event)
				go func() {
					<-ctx.Done() // block until the job is cancelled
					close(c)
				}()
				return c
			}),
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), gomock.Len(0), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				cancel() // all good, stop Run()
				c := make(chan event.Event)
				close(c)
				return c
			}),
	)
	w.Run(ctx)
}

func TestRun_ApplyIsRetriedOnError(t *testing.T) {
	w, applier, watcher := setupWorker(t)
	w.applierBackoff = retry.NewExponentialBackoffFactory(time.Millisecond, time.Minute, time.Minute, 2, 1)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.project.Paths,
	}
	watcher.EXPECT().
		Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
		Do(func(ctx context.Context, req *rpc.ObjectsToSynchronizeRequest, callback rpc.ObjectsToSynchronizeCallback) {
			callback(ctx, rpc.ObjectsToSynchronizeData{
				CommitId: revision,
			})
			<-ctx.Done()
		})
	gomock.InOrder(
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), gomock.Len(0), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				c := make(chan event.Event, 1)
				c <- event.Event{
					Type: event.ErrorType,
					ErrorEvent: event.ErrorEvent{
						Err: errors.New("expected error"),
					},
				}
				close(c)
				return c
			}),
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), gomock.Len(0), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				cancel() // all good, stop Run()
				c := make(chan event.Event)
				close(c)
				return c
			}),
	)
	w.Run(ctx)
}

func TestRun_PeriodicApply(t *testing.T) {
	w, applier, watcher := setupWorker(t)
	w.reapplyInterval = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := &rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     w.project.Paths,
	}
	watcher.EXPECT().
		Watch(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
		Do(func(ctx context.Context, req *rpc.ObjectsToSynchronizeRequest, callback rpc.ObjectsToSynchronizeCallback) {
			callback(ctx, rpc.ObjectsToSynchronizeData{
				CommitId: revision,
			})
			<-ctx.Done()
		})
	gomock.InOrder(
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), gomock.Len(0), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				c := make(chan event.Event)
				close(c)
				return c
			}),
		applier.EXPECT().
			Run(gomock.Any(), gomock.Any(), gomock.Len(0), gomock.Any()).
			DoAndReturn(func(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event {
				cancel() // all good, stop Run()
				c := make(chan event.Event)
				close(c)
				return c
			}),
	)
	w.Run(ctx)
}

func setupWorker(t *testing.T) (*defaultGitopsWorker, *MockApplier, *mock_rpc.MockObjectsToSynchronizeWatcherInterface) {
	ctrl := gomock.NewController(t)
	applier := NewMockApplier(ctrl)
	watcher := mock_rpc.NewMockObjectsToSynchronizeWatcherInterface(ctrl)
	w := &defaultGitopsWorker{
		objWatcher: watcher,
		synchronizerConfig: synchronizerConfig{
			log: zaptest.NewLogger(t),
			project: &agentcfg.ManifestProjectCF{
				Id:               projectId,
				DefaultNamespace: defaultNamespace, // as if user didn't specify configuration so it's the default value
				Paths: []*agentcfg.PathCF{
					{
						Glob: "*.yaml",
					},
				},
			},
			applier:         applier,
			k8sUtilFactory:  cmdtesting.NewTestFactory(),
			reapplyInterval: time.Minute,
			applierBackoff:  testhelpers.NewBackoff()(),
		},
	}
	return w, applier, watcher
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

func invObject(id, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "inventory-" + id,
				"namespace": namespace,
				"labels": map[string]interface{}{
					common.InventoryLabel: id,
				},
			},
		},
	}
}
