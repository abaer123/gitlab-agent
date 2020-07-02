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
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/testing/mock_engine"
)

const (
	projectId = "bla123/bla-1"
	namespace = ""
	revision  = "rev12341234"

	twoConfigMaps = `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demomap1
  namespace: test1
data:
  key1: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demomap2
  namespace: test2
data:
  key2: value2
`
	configMap = `apiVersion: v1
kind: ConfigMap
metadata:
  name: demomap3
  namespace: test3
data:
  key3: value3
`
)

func TestRunReturnsNoErrorWhenGetObjectsToSynchronizeFails(t *testing.T) {
	s, _, client, _ := setup(t, false)
	client.EXPECT().
		GetObjectsToSynchronize(gomock.Any(), projectIdMatcher{projectId: projectId}, gomock.Any()).
		Return(nil, errors.New("bla"))

	s.run()
}

func TestRunReturnsNoErrorWhenGetObjectsToSynchronizeRecvFails(t *testing.T) {
	s, _, _, stream := setup(t, true)
	stream.EXPECT().
		Recv().
		Return(nil, errors.New("bla"))

	s.run()
}

func TestRunHappyPathNoObjects(t *testing.T) {
	s, engine, _, stream := setup(t, true)

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
		Sync(gomock.Any(), gomock.Len(0), gomock.Any(), gomock.Eq(revision), gomock.Eq(namespace)).
		Return([]common.ResourceSyncResult{}, nil)

	s.run()
}

func TestRunHappyPath(t *testing.T) {
	s, engine, _, stream := setup(t, true)

	resp := &agentrpc.ObjectsToSynchronizeResponse{
		Revision: revision,
		Objects: []*agentrpc.ObjectToSynchronize{
			{
				// Multi-document YAML
				Object: []byte(twoConfigMaps),
			},
			{
				// Single object
				Object: []byte(configMap),
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
		Sync(gomock.Any(), gomock.Len(3), gomock.Any(), gomock.Eq(revision), gomock.Eq(namespace)).
		Return([]common.ResourceSyncResult{}, nil)

	s.run()
}

func setup(t *testing.T, returnStream bool) (*synchronizer, *mock_engine.MockGitOpsEngine, *mock_agentrpc.MockGitLabServiceClient, *mock_agentrpc.MockGitLabService_GetObjectsToSynchronizeClient) {
	mockCtrl := gomock.NewController(t)
	engine := mock_engine.NewMockGitOpsEngine(mockCtrl)
	client := mock_agentrpc.NewMockGitLabServiceClient(mockCtrl)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s := &synchronizer{
		ctx: ctx,
		eng: engine,
		synchronizerConfig: synchronizerConfig{
			log:       logrus.New().WithFields(nil),
			projectId: projectId,
			namespace: namespace,
			client:    client,
		},
	}
	var stream *mock_agentrpc.MockGitLabService_GetObjectsToSynchronizeClient
	if returnStream {
		stream = mock_agentrpc.NewMockGitLabService_GetObjectsToSynchronizeClient(mockCtrl)
		client.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), projectIdMatcher{projectId: projectId}, gomock.Any()).
			Return(stream, nil)
	}
	return s, engine, client, stream
}

type projectIdMatcher struct {
	projectId string
}

func (e projectIdMatcher) Matches(x interface{}) bool {
	req, ok := x.(*agentrpc.ObjectsToSynchronizeRequest)
	if !ok {
		return false
	}
	return req.ProjectId == e.projectId
}

func (e projectIdMatcher) String() string {
	return fmt.Sprintf("has project id %s", e.projectId)
}
