package agentrpc_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
)

var (
	_ agentrpc.ObjectsToSynchronizeWatcherInterface = &agentrpc.ObjectsToSynchronizeWatcher{}
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
	client := mock_agentrpc.NewMockKasClient(mockCtrl)
	stream1 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	stream2 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
	req := &agentrpc.ObjectsToSynchronizeRequest{
		ProjectId: projectId,
		Paths:     pathsCfg,
	}
	gomock.InOrder(
		client.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, req)).
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
		client.EXPECT().
			GetObjectsToSynchronize(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ObjectsToSynchronizeRequest{
				ProjectId: projectId,
				CommitId:  revision,
				Paths:     pathsCfg,
			})).
			Return(stream2, nil),
		stream2.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ObjectsToSynchronizeResponse, error) {
				cancel()
				return nil, io.EOF
			}),
	)
	w := agentrpc.ObjectsToSynchronizeWatcher{
		Log:         zaptest.NewLogger(t),
		KasClient:   client,
		RetryPeriod: 10 * time.Millisecond,
	}
	w.Watch(ctx, req, func(ctx context.Context, data agentrpc.ObjectsToSynchronizeData) {
		// Don't care
	})
}

func TestObjectsToSynchronizeWatcherInvalidStream(t *testing.T) {
	tests := []struct {
		name   string
		stream []*agentrpc.ObjectsToSynchronizeResponse
		eof    bool
	}{
		{
			name: "empty stream",
			eof:  true,
		},
		{
			name: "missing headers",
			stream: []*agentrpc.ObjectsToSynchronizeResponse{
				{
					Message: &agentrpc.ObjectsToSynchronizeResponse_Trailers_{
						Trailers: &agentrpc.ObjectsToSynchronizeResponse_Trailers{},
					},
				},
			},
		},
		{
			name: "unexpected headers",
			stream: []*agentrpc.ObjectsToSynchronizeResponse{
				{
					Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
						Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
							CommitId: revision,
						},
					},
				},
				{
					Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
						Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
							CommitId: revision,
						},
					},
				},
			},
		},
		{
			name: "missing trailers",
			stream: []*agentrpc.ObjectsToSynchronizeResponse{
				{
					Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
						Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
							CommitId: revision,
						},
					},
				},
			},
			eof: true,
		},
		{
			name: "trailers then headers",
			stream: []*agentrpc.ObjectsToSynchronizeResponse{
				{
					Message: &agentrpc.ObjectsToSynchronizeResponse_Trailers_{
						Trailers: &agentrpc.ObjectsToSynchronizeResponse_Trailers{},
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
			client := mock_agentrpc.NewMockKasClient(mockCtrl)
			stream1 := mock_agentrpc.NewMockKas_GetObjectsToSynchronizeClient(mockCtrl)
			req := &agentrpc.ObjectsToSynchronizeRequest{
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
					calls = append(calls, stream1.EXPECT().Recv().Return(streamItem, nil))
				}
				calls = append(calls, stream1.EXPECT().
					Recv().
					DoAndReturn(func() (*agentrpc.ObjectsToSynchronizeResponse, error) {
						cancel()
						return nil, io.EOF
					}))
			} else {
				for i := 0; i < len(tc.stream)-1; i++ { // nolint:scopelint
					streamItem := tc.stream[i] // nolint:scopelint
					calls = append(calls, stream1.EXPECT().Recv().Return(streamItem, nil))
				}
				calls = append(calls, stream1.EXPECT().Recv().DoAndReturn(func() (*agentrpc.ObjectsToSynchronizeResponse, error) {
					cancel()
					return tc.stream[len(tc.stream)-1], nil // nolint:scopelint
				}))
			}
			gomock.InOrder(calls...)
			w := agentrpc.ObjectsToSynchronizeWatcher{
				Log:         zaptest.NewLogger(t),
				KasClient:   client,
				RetryPeriod: 10 * time.Millisecond,
			}
			w.Watch(ctx, req, func(ctx context.Context, data agentrpc.ObjectsToSynchronizeData) {
				// Must not be called
				t.FailNow()
			})
		})
	}
}
