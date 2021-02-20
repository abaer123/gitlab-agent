package rpc_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
)

var (
	_ rpc.ObjectsToSynchronizeWatcherInterface = &rpc.ObjectsToSynchronizeWatcher{}
)

const (
	projectId = "bla123/bla-1"
	revision  = "rev12341234"
)

func TestObjectsToSynchronizeWatcherResumeConnection(t *testing.T) {
	pathsCfg := []*agentcfg.PathCF{
		{
			Glob: "*.yaml",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	client := mock_rpc.NewMockGitopsClient(mockCtrl)
	stream1 := mock_rpc.NewMockGitops_GetObjectsToSynchronizeClient(mockCtrl)
	stream2 := mock_rpc.NewMockGitops_GetObjectsToSynchronizeClient(mockCtrl)
	req := &rpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     pathsCfg,
	}
	gomock.InOrder(
		client.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, req)).
			Return(stream1, nil),
		stream1.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Header_{
					Header: &rpc.ObjectsToSynchronizeResponse_Header{
						CommitId: revision,
					},
				},
			})),
		stream1.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Trailers_{
					Trailers: &rpc.ObjectsToSynchronizeResponse_Trailers{},
				},
			})),
		stream1.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF),
		client.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				CommitId:  revision,
				Paths:     pathsCfg,
			})).
			Return(stream2, nil),
		stream2.EXPECT().
			RecvMsg(gomock.Any()).
			DoAndReturn(func(msg interface{}) error {
				cancel()
				return io.EOF
			}),
	)
	w := rpc.ObjectsToSynchronizeWatcher{
		Log:          zaptest.NewLogger(t),
		GitopsClient: client,
		RetryPeriod:  10 * time.Millisecond,
	}
	w.Watch(ctx, req, func(ctx context.Context, data rpc.ObjectsToSynchronizeData) {
		// Don't care
	})
}

func TestObjectsToSynchronizeWatcherInvalidStream(t *testing.T) {
	tests := []struct {
		name   string
		stream []*rpc.ObjectsToSynchronizeResponse
		eof    bool
	}{
		{
			name: "empty stream",
			eof:  true,
		},
		{
			name: "missing header",
			stream: []*rpc.ObjectsToSynchronizeResponse{
				{
					Message: &rpc.ObjectsToSynchronizeResponse_Trailers_{
						Trailers: &rpc.ObjectsToSynchronizeResponse_Trailers{},
					},
				},
			},
		},
		{
			name: "unexpected header",
			stream: []*rpc.ObjectsToSynchronizeResponse{
				{
					Message: &rpc.ObjectsToSynchronizeResponse_Header_{
						Header: &rpc.ObjectsToSynchronizeResponse_Header{
							CommitId: revision,
						},
					},
				},
				{
					Message: &rpc.ObjectsToSynchronizeResponse_Header_{
						Header: &rpc.ObjectsToSynchronizeResponse_Header{
							CommitId: revision,
						},
					},
				},
			},
		},
		{
			name: "missing trailers",
			stream: []*rpc.ObjectsToSynchronizeResponse{
				{
					Message: &rpc.ObjectsToSynchronizeResponse_Header_{
						Header: &rpc.ObjectsToSynchronizeResponse_Header{
							CommitId: revision,
						},
					},
				},
			},
			eof: true,
		},
		{
			name: "trailers then header",
			stream: []*rpc.ObjectsToSynchronizeResponse{
				{
					Message: &rpc.ObjectsToSynchronizeResponse_Trailers_{
						Trailers: &rpc.ObjectsToSynchronizeResponse_Trailers{},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mockCtrl := gomock.NewController(t)
			client := mock_rpc.NewMockGitopsClient(mockCtrl)
			stream1 := mock_rpc.NewMockGitops_GetObjectsToSynchronizeClient(mockCtrl)
			req := &rpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				Paths: []*agentcfg.PathCF{
					{
						Glob: "*.yaml",
					},
				},
			}
			calls := []*gomock.Call{
				client.EXPECT().
					GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, req)).
					Return(stream1, nil),
			}
			if tc.eof { // nolint:scopelint
				for _, streamItem := range tc.stream { // nolint:scopelint
					calls = append(calls, stream1.EXPECT().
						RecvMsg(gomock.Any()).
						Do(testhelpers.RecvMsg(streamItem)),
					)
				}
				calls = append(calls, stream1.EXPECT().
					RecvMsg(gomock.Any()).
					DoAndReturn(func(msg interface{}) error {
						cancel()
						return io.EOF
					}))
			} else {
				for i := 0; i < len(tc.stream)-1; i++ { // nolint:scopelint
					streamItem := tc.stream[i] // nolint:scopelint
					calls = append(calls, stream1.EXPECT().
						RecvMsg(gomock.Any()).
						Do(testhelpers.RecvMsg(streamItem)),
					)
				}
				calls = append(calls, stream1.EXPECT().RecvMsg(gomock.Any()).Do(func(msg interface{}) {
					testhelpers.SetValue(msg, tc.stream[len(tc.stream)-1]) // nolint:scopelint
					cancel()
				}))
			}
			gomock.InOrder(calls...)
			w := rpc.ObjectsToSynchronizeWatcher{
				Log:          zaptest.NewLogger(t),
				GitopsClient: client,
				RetryPeriod:  10 * time.Millisecond,
			}
			w.Watch(ctx, req, func(ctx context.Context, data rpc.ObjectsToSynchronizeData) {
				// Must not be called
				t.FailNow()
			})
		})
	}
}
