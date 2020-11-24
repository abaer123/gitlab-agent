package gitops_agent

import (
	"bytes"
	"context"

	"github.com/argoproj/gitops-engine/pkg/engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	managedObjectAnnotationName = "k8s-agent.gitlab.com/managed-object"
)

// desiredState holds the state to put into a Kubernetes cluster.
type desiredState struct {
	commitId string
	sources  []stateSource
}

type stateSource struct {
	name string
	data []byte
}

// synchronizerConfig holds configuration for a synchronizer.
type synchronizerConfig struct {
	log                  *zap.Logger
	projectConfiguration *agentcfg.ManifestProjectCF
	k8sClientGetter      resource.RESTClientGetter
}

type resourceInfo struct {
	gcMark string
}

type synchronizer struct {
	synchronizerConfig
	engine       engine.GitOpsEngine
	desiredState chan desiredState
}

func newSynchronizer(config synchronizerConfig, engine engine.GitOpsEngine) *synchronizer {
	return &synchronizer{
		synchronizerConfig: config,
		engine:             engine,
		desiredState:       make(chan desiredState),
	}
}

func (s *synchronizer) setDesiredState(ctx context.Context, state desiredState) bool {
	select {
	case <-ctx.Done():
		return false
	case s.desiredState <- state:
		return true
	}
}

func (s *synchronizer) run(ctx context.Context) {
	jobs := make(chan syncJob)
	sw := newSyncWorker(s.synchronizerConfig, s.engine)
	var wg wait.Group
	defer wg.Wait()   // Wait for sw to exit
	defer close(jobs) // Close jobs to signal sw there is no more work to be done
	wg.Start(func() {
		sw.run(jobs) // Start sw
	})

	var (
		jobsCh    chan syncJob
		newJob    syncJob
		jobCancel context.CancelFunc
	)
	defer func() {
		if jobCancel != nil {
			jobCancel()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return // nolint: govet
		case state := <-s.desiredState:
			objs, err := s.decodeObjectsToSynchronize(state.sources)
			if err != nil {
				s.log.Warn("Failed to decode GitOps objects", zap.Error(err))
				continue
			}
			if jobCancel != nil {
				jobCancel() // Cancel running/pending job ASAP
			}
			markAsManaged(objs)
			newJob = syncJob{
				commitId: state.commitId,
				objects:  objs,
			}
			newJob.ctx, jobCancel = context.WithCancel(context.Background()) // nolint: govet
			jobsCh = jobs                                                    // Enable select case
		case jobsCh <- newJob: // Try to send new job to syncWorker. This case is active only when jobsCh is not nil
			// Success!
			newJob = syncJob{} // Erase contents to help GC
			jobsCh = nil       // Disable this select case (send to nil channel blocks forever)
		}
	}
}

func (s *synchronizer) decodeObjectsToSynchronize(sources []stateSource) ([]*unstructured.Unstructured, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	// TODO allow enforcing namespace
	builder := resource.NewBuilder(s.k8sClientGetter).
		ContinueOnError().
		Flatten().
		Local().
		Unstructured()
	for _, source := range sources {
		builder.Stream(bytes.NewReader(source.data), source.name)
	}
	var res []*unstructured.Unstructured
	err := builder.Do().Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		un := info.Object.(*unstructured.Unstructured)
		// TODO enforce namespace is set for namespaced objects?
		res = append(res, un)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func markAsManaged(objs []*unstructured.Unstructured) {
	for _, obj := range objs {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string, 1)
		}
		annotations[managedObjectAnnotationName] = "managed" // TODO
		obj.SetAnnotations(annotations)
	}
}

func populateResourceInfoHandler(un *unstructured.Unstructured, isRoot bool) (interface{} /*info*/, bool /*cacheManifest*/) {
	// store gc mark of every resource
	gcMark := un.GetAnnotations()[managedObjectAnnotationName]
	// cache resources that has that mark to improve performance
	return &resourceInfo{
		gcMark: gcMark,
	}, gcMark != ""
}
