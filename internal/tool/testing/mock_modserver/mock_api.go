package mock_modserver

import (
	"github.com/golang/mock/gomock"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"google.golang.org/grpc"
)

func NewMockAPIWithMockPoller(ctrl *gomock.Controller, pollTimes int) *MockAPI {
	mockApi := NewMockAPI(ctrl)
	if pollTimes > 0 {
		mockApi.EXPECT().
			PollWithBackoff(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(stream grpc.ServerStream, cfg retry.PollConfig, f retry.PollWithBackoffFunc) error {
				for i := 0; i < pollTimes; i++ {
					err, res := f()
					if res == retry.Done {
						return err
					}
				}
				return nil
			})
	}
	return mockApi
}
