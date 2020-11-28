package mock_modserver

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

func NewMockAPIWithMockPoller(ctrl *gomock.Controller, pollTimes int) *MockAPI {
	mockApi := NewMockAPI(ctrl)
	if pollTimes > 0 {
		mockApi.EXPECT().
			PollImmediateUntil(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, interval, connectionMaxAge time.Duration, condition modserver.ConditionFunc) error {
				for i := 0; i < pollTimes; i++ {
					done, err := condition()
					if err != nil || done {
						return err
					}
				}
				return nil
			})
	}
	return mockApi
}
