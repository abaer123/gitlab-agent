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
	typed_v2 "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/typed/cilium.io/v2"
	monitor_api "github.com/cilium/cilium/pkg/monitor/api"
	legacy_proto "github.com/golang/protobuf/proto" // nolint:staticcheck
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	getFlowsRetryPeriod = 10 * time.Second
)

type worker struct {
	log            *zap.Logger
	api            modagent.API
	ciliumClient   typed_v2.CiliumV2Interface
	observerClient observer.ObserverClient
	projectId      int64
}

func (w *worker) Run(ctx context.Context) {
	retry.JitterUntil(ctx, getFlowsRetryPeriod, func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		// L4 https://gitlab.com/gitlab-org/gitlab/-/issues/293931
		// L7 https://gitlab.com/gitlab-org/gitlab/-/issues/293932
		flc, err := w.observerClient.GetFlows(ctx, &observer.GetFlowsRequest{
			Follow: true,
			Whitelist: []*flow.FlowFilter{
				{
					Verdict: []flow.Verdict{flow.Verdict_DROPPED}, //Drop verdicts only
					EventType: []*flow.EventTypeFilter{
						{
							Type: monitor_api.MessageTypePolicyVerdict, //L3_L4
						},
					},
				},
			},
			Since: timestamppb.Now(),
		})
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				w.log.Error("failed to get flows from hubble relay", zap.Error(err))
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
				err = w.processFlow(ctx, value.Flow)
				if err != nil {
					w.log.Error("flow processing failed", zap.Error(err))
					return
				}
			}
		}
	})
}

func (w *worker) processFlow(ctx context.Context, flw *flow.Flow) error {
	ns := getNamespace(flw)
	if ns == "" {
		return nil
	}
	cnps, err := w.ciliumClient.CiliumNetworkPolicies(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.gitlab.com/proj": strconv.FormatInt(w.projectId, 10),
			},
		}),
	})
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
		w.log.Info("successfull response when creating alerts from cilium_alert endpoint", zap.Int32("status_code", resp.StatusCode))
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
