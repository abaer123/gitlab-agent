package agentk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/testing/kube_testing"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/testing/mock_engine"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	projectId = "bla123/bla-1"
	namespace = ""
	revision  = "rev12341234"
)

func TestRunReturnsNoErrorWhenGetObjectsToSynchronizeFails(t *testing.T) {
	s, _, client, _ := setupSynchronizer(t, false)
	client.EXPECT().
		GetObjectsToSynchronize(gomock.Any(), projectIDMatcher{projectId: projectId}, gomock.Any()).
		Return(nil, errors.New("bla"))

	s.run()
}

func TestRunReturnsNoErrorWhenGetObjectsToSynchronizeRecvFails(t *testing.T) {
	s, _, _, stream := setupSynchronizer(t, true)
	stream.EXPECT().
		Recv().
		Return(nil, errors.New("bla"))

	s.run()
}

func TestRunHappyPathNoObjects(t *testing.T) {
	s, engine, _, stream := setupSynchronizer(t, true)

	resp := &agentrpc.ObjectsToSynchronizeResponse{
		Revision: revision,
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
		Sync(gomock.Any(), gomock.Len(0), gomock.Any(), revision, namespace).
		Return([]common.ResourceSyncResult{}, nil)

	s.run()
}

func TestRunHappyPath(t *testing.T) {
	s, engine, _, stream := setupSynchronizer(t, true)
	objs := []*unstructured.Unstructured{
		kube_testing.ToUnstructured(t, testNs1()),
		kube_testing.ToUnstructured(t, testMap1()),
		kube_testing.ToUnstructured(t, testMap2()),
	}
	resp := &agentrpc.ObjectsToSynchronizeResponse{
		Revision: revision,
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
	gomock.InOrder(
		stream.EXPECT().
			Recv().
			Return(resp, nil),
		stream.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	engine.EXPECT().
		Sync(gomock.Any(), matcher.K8sObjectEq(t, objs, kube_testing.IgnoreAnnotation(managedObjectAnnotationName)), gomock.Any(), revision, namespace, gomock.Any()).
		Return([]common.ResourceSyncResult{}, nil)

	s.run()
}

func setupSynchronizer(t *testing.T, returnStream bool) (*synchronizer, *mock_engine.MockGitOpsEngine, *mock_agentrpc.MockKasClient, *mock_agentrpc.MockKas_GetObjectsToSynchronizeClient) {
	mockCtrl := gomock.NewController(t)
	engine := mock_engine.NewMockGitOpsEngine(mockCtrl)
	client := mock_agentrpc.NewMockKasClient(mockCtrl)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	configFlags := genericclioptions.NewTestConfigFlags()
	s := &synchronizer{
		ctx: ctx,
		eng: engine,
		synchronizerConfig: synchronizerConfig{
			log:             logrus.New().WithFields(nil),
			projectId:       projectId,
			namespace:       namespace,
			kasClient:       client,
			k8sClientGetter: configFlags,
		},
	}
	var stream *mock_agentrpc.MockKas_GetObjectsToSynchronizeClient
	if returnStream {
		stream = mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
		client.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), projectIDMatcher{projectId: projectId}, gomock.Any()).
			Return(stream, nil)
	}
	return s, engine, client, stream
}

type projectIDMatcher struct {
	projectId string
}

func (e projectIDMatcher) Matches(x interface{}) bool {
	req, ok := x.(*agentrpc.ObjectsToSynchronizeRequest)
	if !ok {
		return false
	}
	return req.ProjectId == e.projectId
}

func (e projectIDMatcher) String() string {
	return fmt.Sprintf("has project id %s", e.projectId)
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
