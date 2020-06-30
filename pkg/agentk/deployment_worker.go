package agentk

import (
	"context"
	"io"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

const (
	engineRunRetryPeriod               = 10 * time.Second
	getObjectsToSynchronizeRetryPeriod = 10 * time.Second
)

var (
	yamlSerializer = json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		unstructuredscheme.NewUnstructuredCreator(),
		unstructuredscheme.NewUnstructuredObjectTyper(),
		json.SerializerOptions{Yaml: true})
)

type deploymentWorker struct {
	kubeClientConfig *rest.Config
	synchronizerConfig
}

func (d *deploymentWorker) Run(ctx context.Context) {
	clusterCache := cache.NewClusterCache(d.kubeClientConfig,
		cache.SetPopulateResourceInfoHandler(populateResourceInfoHandler))
	eng := engine.NewEngine(d.kubeClientConfig, clusterCache)
	var stopEngine io.Closer
	err := wait.PollImmediateUntil(engineRunRetryPeriod, func() (bool /*done*/, error) {
		var err error
		stopEngine, err = eng.Run()
		if err != nil {
			d.log.WithError(err).Warn("engine.Run() failed")
			return false, nil // nil error to keep polling
		}
		return true, nil
	}, ctx.Done())
	if err != nil {
		// context is done
		return
	}
	defer stopEngine.Close()

	s := synchronizer{
		ctx:                ctx,
		eng:                eng,
		synchronizerConfig: d.synchronizerConfig,
	}

	_ = wait.PollImmediateUntil(getObjectsToSynchronizeRetryPeriod, func() (bool /*done*/, error) {
		s.run()
		return false, nil // never done, never error. Polling is interrupted by ctx
	}, ctx.Done())
}
