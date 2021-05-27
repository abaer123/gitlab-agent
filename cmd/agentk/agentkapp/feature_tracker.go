package agentkapp

import (
	"sync"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"go.uber.org/zap"
)

type featureTracker struct {
	mx          sync.Mutex
	log         *zap.Logger
	status      map[modagent.Feature]map[string]struct{} // feature -> set of consumers
	subscribers map[modagent.Feature][]modagent.SubscribeCb
}

func newFeatureTracker(log *zap.Logger) *featureTracker {
	return &featureTracker{
		log:         log,
		status:      map[modagent.Feature]map[string]struct{}{},
		subscribers: map[modagent.Feature][]modagent.SubscribeCb{},
	}
}

func (f *featureTracker) ToggleFeature(feature modagent.Feature, consumer string, enabled bool) {
	f.mx.Lock()
	defer f.mx.Unlock()
	notify := false
	status := f.status[feature]
	if enabled {
		if _, ok := status[consumer]; ok {
			// Already enabled by this consumer, nothing to do
			return
		}
		// Not enabled by this consumer, need to record that
		if status == nil {
			status = map[string]struct{}{}
			f.status[feature] = status
			notify = true // first consumer to enable, must notify
		}
		status[consumer] = struct{}{}
	} else {
		if _, ok := status[consumer]; !ok {
			// Already disabled by this consumer, nothing to do
			return
		}
		delete(status, consumer)
		if len(status) == 0 {
			delete(f.status, feature)
			notify = true // last consumer disabled, must notify
		}
	}
	if notify {
		f.log.Info("Feature status change", featureName(feature), featureStatus(enabled))
		for _, cb := range f.subscribers[feature] {
			cb(enabled)
		}
	}
}

func (f *featureTracker) Subscribe(feature modagent.Feature, cb modagent.SubscribeCb) {
	f.mx.Lock()
	defer f.mx.Unlock()
	f.subscribers[feature] = append(f.subscribers[feature], cb)
}
