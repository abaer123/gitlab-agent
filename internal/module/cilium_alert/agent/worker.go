package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cilium/cilium/api/v1/flow"
	"github.com/cilium/cilium/api/v1/observer"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	informers_v2 "github.com/cilium/cilium/pkg/k8s/client/informers/externalversions/cilium.io/v2"
	monitor_api "github.com/cilium/cilium/pkg/monitor/api"
	legacy_proto "github.com/golang/protobuf/proto" // nolint:staticcheck
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
)

const (
	getFlowsRetryPeriod   = 10 * time.Second
	informerResyncPeriod  = 30 * time.Minute
	informerNotifierIndex = "InformerNotifierIdx"
	gitLabProjectLabel    = "app.gitlab.com/proj"
)

type worker struct {
	log            *zap.Logger
	api            modagent.API
	ciliumClient   versioned.Interface
	observerClient observer.ObserverClient
	projectId      int64
}

func cnpIndexFunc(obj interface{}) ([]string, error) {
	cnp := obj.(*v2.CiliumNetworkPolicy)

	if alertsEnabled := cnp.Annotations[alertAnnotationKey]; alertsEnabled != "true" {
		return nil, nil
	}

	projectId, ok := cnp.Labels[gitLabProjectLabel]
	if !ok {
		return nil, nil
	}

	return []string{projectId}, nil
}

func (w *worker) Run(ctx context.Context) {
	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      gitLabProjectLabel,
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	})

	ciliumEndpointInformer := informers_v2.NewFilteredCiliumNetworkPolicyInformer(
		w.ciliumClient,
		metav1.NamespaceNone,
		informerResyncPeriod,
		cache.Indexers{informerNotifierIndex: cnpIndexFunc},
		func(listOptions *metav1.ListOptions) { listOptions.LabelSelector = labelSelector },
	)

	var wg wait.Group
	defer wg.Wait()
	wg.StartWithChannel(ctx.Done(), ciliumEndpointInformer.Run)

	if !cache.WaitForCacheSync(ctx.Done(), ciliumEndpointInformer.HasSynced) {
		return
	}

	retry.JitterUntil(ctx, getFlowsRetryPeriod, func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		flc, err := w.observerClient.GetFlows(ctx, &observer.GetFlowsRequest{
			Follow: true,
			Whitelist: []*flow.FlowFilter{
				{
					Verdict: []flow.Verdict{flow.Verdict_DROPPED}, //Drop verdicts only
					EventType: []*flow.EventTypeFilter{
						{
							Type: monitor_api.MessageTypePolicyVerdict, //L3_L4
						},
						{
							Type: monitor_api.MessageTypeAccessLog, //L7
						},
					},
				},
			},
			Since: timestamppb.Now(),
		})
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				w.log.Error("Failed to get flows from Hubble relay", zap.Error(err))
			}
			return
		}
		for {
			resp, err := flc.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				if !grpctool.RequestCanceled(err) {
					w.log.Error("GetFlows.Recv() failed", zap.Error(err))
				}
				return
			}
			switch value := resp.ResponseTypes.(type) {
			case *observer.GetFlowsResponse_Flow:
				err = w.processFlow(ctx, value.Flow, ciliumEndpointInformer)
				if err != nil {
					w.log.Error("Flow processing failed", zap.Error(err))
					return
				}
			}
		}
	})
}

func (w *worker) processFlow(ctx context.Context, flw *flow.Flow, informer cache.SharedIndexInformer) error {
	cnps, err := informer.GetIndexer().ByIndex(informerNotifierIndex, strconv.FormatInt(w.projectId, 10))
	if err != nil {
		return err
	}

	cnp, err := getPolicy(flw, cnps)
	if err != nil {
		return err
	}
	if cnp == nil {
		return nil
	}
	return w.sendAlert(ctx, flw, cnp)
}

func (w *worker) sendAlert(ctx context.Context, fl *flow.Flow, cnp *v2.CiliumNetworkPolicy) error {
	mbdy, err := json.Marshal(payload{
		Alert: alert{
			Flow:                (*flowAlias)(fl),
			CiliumNetworkPolicy: cnp,
		},
	})
	if err != nil {
		return fmt.Errorf("failed while encapsulating alert: %v", err)
	}
	resp, err := w.api.MakeGitLabRequest(ctx, "/",
		modagent.WithRequestMethod(http.MethodPost),
		modagent.WithRequestHeader("Content-Type", "application/json"),
		modagent.WithRequestBody(bytes.NewReader(mbdy)),
	)
	if err != nil {
		return fmt.Errorf("failed request to internal api: %v", err)
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		w.log.Info("successful response when creating alerts from cilium_alert endpoint", zap.Int32("status_code", resp.StatusCode))
	default:
		return fmt.Errorf("failed to send flow to internal API: got %d HTTP response code", resp.StatusCode)
	}
	return nil
}

var (
	_ json.Marshaler   = &flowAlias{}
	_ json.Unmarshaler = &flowAlias{}
)

type flowAlias flow.Flow

func (f *flowAlias) MarshalJSON() ([]byte, error) {
	typedF := (*flow.Flow)(f)
	return protojson.Marshal(legacy_proto.MessageV2(typedF))
}

func (f *flowAlias) UnmarshalJSON(data []byte) error {
	typedF := (*flow.Flow)(f)
	return protojson.Unmarshal(data, legacy_proto.MessageV2(typedF))
}

type payload struct {
	Alert alert `json:"alert"`
}

type alert struct {
	Flow                *flowAlias              `json:"flow"`
	CiliumNetworkPolicy *v2.CiliumNetworkPolicy `json:"ciliumNetworkPolicy"`
}
